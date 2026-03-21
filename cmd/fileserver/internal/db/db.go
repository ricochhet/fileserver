package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	sqlite "modernc.org/sqlite"
)

const defaultDbPath = "fileserver.db"

var (
	ErrUserNotFound       = errors.New("db: user not found")
	ErrInvalidCredentials = errors.New("db: invalid credentials")
)

// DB wraps a SQLite connection and exposes typed operations used by the server.
type DB struct {
	conn *sql.DB
}

// User is a stored user account.
type User struct {
	Username    string
	Password    string
	DisplayName string
}

// Message is a stored chat message.
type Message struct {
	ID          string
	ChannelCode string
	ChannelName string
	Username    string
	DisplayName string
	Body        string
	Timestamp   time.Time
}

// connector implements driver.Connector for sqlite.Driver, enabling sql.OpenDB
// without relying on the global driver registry (and thus a blank import).
type connector struct {
	drv  *sqlite.Driver
	name string
}

func (c *connector) Connect(_ context.Context) (driver.Conn, error) {
	return c.drv.Open(c.name)
}

func (c *connector) Driver() driver.Driver {
	return c.drv
}

// Open opens or creates the SQLite database at path, applies pragmas, and runs migrations.
func Open(path string) (*DB, error) {
	conn := sql.OpenDB(&connector{drv: &sqlite.Driver{}, name: path})

	ctx := context.Background()

	// WAL allows concurrent readers alongside the single writer without blocking on every write.
	if _, err := conn.ExecContext(ctx, queryPragmas); err != nil {
		conn.Close()
		return nil, fmt.Errorf("db.Open pragma: %w", err)
	}

	d := &DB{conn: conn}
	if err := d.migrate(ctx); err != nil {
		conn.Close()
		return nil, fmt.Errorf("db.Open migrate: %w", err)
	}

	return d, nil
}

// Close closes the underlying database connection.
func (d *DB) Close() error {
	return d.conn.Close()
}

// UpsertUser inserts or updates a user record, keeping config credentials in sync on each startup.
func (d *DB) UpsertUser(ctx context.Context, username, password, displayName string) error {
	_, err := d.conn.ExecContext(ctx, queryUpsertUser, username, password, displayName)

	return err
}

// GetUser returns the user with the given username, or ErrUserNotFound if no row exists.
func (d *DB) GetUser(ctx context.Context, username string) (*User, error) {
	u := &User{}
	err := d.conn.QueryRowContext(ctx, queryGetUser, username).
		Scan(&u.Username, &u.Password, &u.DisplayName)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}

	if err != nil {
		return nil, err
	}

	return u, nil
}

// AuthUser verifies credentials and returns the matching User, or ErrUserNotFound / ErrInvalidCredentials.
func (d *DB) AuthUser(ctx context.Context, username, password string) (*User, error) {
	u, err := d.GetUser(ctx, username)
	if err != nil {
		return nil, err
	}

	if u.Password != password {
		return nil, ErrInvalidCredentials
	}

	return u, nil
}

// MaxMessageID returns the highest stored message ID, used to seed the sequence counter on startup.
func (d *DB) MaxMessageID(ctx context.Context) (uint64, error) {
	var maxID uint64

	err := d.conn.QueryRowContext(ctx, queryMaxMessageID).Scan(&maxID)

	return maxID, err
}

// SaveMessage persists a message; duplicate IDs are silently ignored.
func (d *DB) SaveMessage(ctx context.Context, m *Message) error {
	_, err := d.conn.ExecContext(ctx, queryInsertMessage,
		m.ID, m.ChannelCode, m.ChannelName,
		m.Username, m.DisplayName, m.Body,
		m.Timestamp.Unix(),
	)

	return err
}

// GetMessages returns up to limit messages for a channel in chronological order.
func (d *DB) GetMessages(ctx context.Context, code string, limit int) ([]*Message, error) {
	rows, err := d.conn.QueryContext(ctx, queryGetMessages, code, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []*Message

	for rows.Next() {
		m := &Message{}

		var ts int64

		if err := rows.Scan(
			&m.ID, &m.ChannelCode, &m.ChannelName,
			&m.Username, &m.DisplayName, &m.Body, &ts,
		); err != nil {
			return nil, err
		}

		m.Timestamp = time.Unix(ts, 0).UTC()
		msgs = append(msgs, m)
	}

	return msgs, rows.Err()
}

// Path returns the path for fileserver.db next to the running executable.
func Path(s string) string {
	path := s
	if s == "" {
		path = defaultDbPath
	}

	exe, err := os.Executable()
	if err != nil {
		return path
	}

	return filepath.Join(filepath.Dir(exe), path)
}

// ListUsers returns all users ordered by username.
func (d *DB) ListUsers(ctx context.Context) ([]*User, error) {
	rows, err := d.conn.QueryContext(ctx, queryListUsers)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*User

	for rows.Next() {
		u := &User{}
		if err := rows.Scan(&u.Username, &u.DisplayName); err != nil {
			return nil, err
		}

		users = append(users, u)
	}

	return users, rows.Err()
}

// DeleteUser removes the user with the given username, returning ErrUserNotFound if absent.
func (d *DB) DeleteUser(ctx context.Context, username string) error {
	res, err := d.conn.ExecContext(ctx, queryDeleteUser, username)
	if err != nil {
		return err
	}

	n, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if n == 0 {
		return ErrUserNotFound
	}

	return nil
}

// migrate creates tables and indices idempotently; new statements can be appended safely.
func (d *DB) migrate(ctx context.Context) error {
	stmts := []string{
		queryCreateUsersTable,
		queryCreateMessagesTable,
		queryCreateMessagesIndex,
	}

	for _, s := range stmts {
		if _, err := d.conn.ExecContext(ctx, s); err != nil {
			return fmt.Errorf("migrate: %w (statement: %.60s…)", err, s)
		}
	}

	return nil
}
