package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ricochhet/fileserver/pkg/strutil"
	sqlite "modernc.org/sqlite"
)

const defaultDbPath = "fileserver.db"

var (
	ErrUserNotFound       = errors.New("db: user not found")
	ErrInvalidCredentials = errors.New("db: invalid credentials")
	ErrChannelNotFound    = errors.New("db: channel not found")
)

type DB struct {
	conn *sql.DB
}

type User struct {
	Username    string
	Password    string
	DisplayName string
	IsAdmin     bool
}

type Channel struct {
	Code string
	Name string
}

type Message struct {
	ID          string
	ChannelCode string
	ChannelName string
	Username    string
	DisplayName string
	Body        string
	Timestamp   time.Time
}

type connector struct {
	drv  *sqlite.Driver
	name string
}

// Connect returns a new connection the database.
func (c *connector) Connect(_ context.Context) (driver.Conn, error) {
	return c.drv.Open(c.name)
}

// Driver returns the sqlite.Driver from the connector.
func (c *connector) Driver() driver.Driver {
	return c.drv
}

// Open opens or creates the SQLite database at path, applies pragmas, and runs migrations.
func Open(path string) (*DB, error) {
	conn := sql.OpenDB(&connector{drv: &sqlite.Driver{}, name: path})

	ctx := context.Background()

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

// UpsertUser inserts or updates a user record.
func (d *DB) UpsertUser(
	ctx context.Context,
	username, password, displayName string,
	isAdmin bool,
) error {
	isAdminInt := 0
	if isAdmin {
		isAdminInt = 1
	}

	_, err := d.conn.ExecContext(ctx, queryUpsertUser, username, password, displayName, isAdminInt)

	return err
}

// InsertUserIfNotExists writes a new user row only when no row with that
// username already exists.
func (d *DB) InsertUserIfNotExists(
	ctx context.Context,
	username, password, displayName string,
	isAdmin bool,
) error {
	isAdminInt := 0
	if isAdmin {
		isAdminInt = 1
	}

	_, err := d.conn.ExecContext(
		ctx,
		queryInsertUserIfNotExists,
		username,
		password,
		displayName,
		isAdminInt,
	)

	return err
}

// GetUser returns the user with the given username, or ErrUserNotFound if no row exists.
func (d *DB) GetUser(ctx context.Context, username string) (*User, error) {
	u := &User{}

	var isAdminInt int

	err := d.conn.QueryRowContext(ctx, queryGetUser, username).
		Scan(&u.Username, &u.Password, &u.DisplayName, &isAdminInt)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}

	if err != nil {
		return nil, err
	}

	u.IsAdmin = isAdminInt != 0

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
	path := strutil.Or(s, defaultDbPath)

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

		var isAdminInt int

		if err := rows.Scan(&u.Username, &u.DisplayName, &isAdminInt); err != nil {
			return nil, err
		}

		u.IsAdmin = isAdminInt != 0
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

// UpsertChannel inserts or updates a channel record.
func (d *DB) UpsertChannel(ctx context.Context, code, name string) error {
	_, err := d.conn.ExecContext(ctx, queryUpsertChannel, code, name)
	return err
}

// GetChannel returns the channel with the given code, or ErrChannelNotFound if absent.
func (d *DB) GetChannel(ctx context.Context, code string) (*Channel, error) {
	ch := &Channel{}
	err := d.conn.QueryRowContext(ctx, queryGetChannel, code).Scan(&ch.Code, &ch.Name)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrChannelNotFound
	}

	if err != nil {
		return nil, err
	}

	return ch, nil
}

// ListChannels returns all channels ordered by code.
func (d *DB) ListChannels(ctx context.Context) ([]*Channel, error) {
	rows, err := d.conn.QueryContext(ctx, queryListChannels)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var channels []*Channel

	for rows.Next() {
		ch := &Channel{}
		if err := rows.Scan(&ch.Code, &ch.Name); err != nil {
			return nil, err
		}

		channels = append(channels, ch)
	}

	return channels, rows.Err()
}

// DeleteChannel removes the channel with the given code, returning ErrChannelNotFound if absent.
func (d *DB) DeleteChannel(ctx context.Context, code string) error {
	res, err := d.conn.ExecContext(ctx, queryDeleteChannel, code)
	if err != nil {
		return err
	}

	n, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if n == 0 {
		return ErrChannelNotFound
	}

	return nil
}

// migrate creates tables and indices idempotently; new statements can be appended safely.
func (d *DB) migrate(ctx context.Context) error {
	stmts := []string{
		queryCreateUsersTable,
		queryCreateChannelsTable,
		queryCreateMessagesTable,
		queryCreateMessagesIndex,
	}

	for _, s := range stmts {
		if _, err := d.conn.ExecContext(ctx, s); err != nil {
			return fmt.Errorf("migrate: %w (statement: %.60s…)", err, s)
		}
	}

	d.migrateAddColumn(ctx, queryMigrateAddIsAdmin)

	return nil
}

// migrateAddColumn executes an ALTER TABLE … ADD COLUMN statement and silently
// ignores the error when the column already exists (SQLite error code 1, message
// contains "duplicate column").
func (d *DB) migrateAddColumn(ctx context.Context, stmt string) {
	if _, err := d.conn.ExecContext(ctx, stmt); err != nil {
		if !strings.Contains(err.Error(), "duplicate column") {
			// Unexpected error — log it but do not abort startup.
			fmt.Printf("db: migrateAddColumn warning: %v\n", err)
		}
	}
}
