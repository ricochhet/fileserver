package db

// Pragma queries.
const (
	queryPragmas = `PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON;`
)

// Migration statements run in order on every startup.
const (
	queryCreateUsersTable = `
CREATE TABLE IF NOT EXISTS users (
	username     TEXT PRIMARY KEY,
	password     TEXT NOT NULL,
	display_name TEXT NOT NULL DEFAULT '',
	is_admin     INTEGER NOT NULL DEFAULT 0
)`

	queryCreateChannelsTable = `
CREATE TABLE IF NOT EXISTS channels (
	code TEXT PRIMARY KEY,
	name TEXT NOT NULL
)`

	queryCreateMessagesTable = `
CREATE TABLE IF NOT EXISTS messages (
	id           TEXT PRIMARY KEY,
	channel_code TEXT NOT NULL,
	channel_name TEXT NOT NULL,
	username     TEXT NOT NULL,
	display_name TEXT NOT NULL,
	body         TEXT NOT NULL,
	timestamp    INTEGER NOT NULL
)`

	queryCreateMessagesIndex = `
CREATE INDEX IF NOT EXISTS idx_messages_channel_ts
	ON messages (channel_code, timestamp DESC)`

	// queryMigrateAddIsAdmin adds the is_admin column to existing databases that
	// pre-date the column.  The caller must tolerate a "duplicate column" error.
	queryMigrateAddIsAdmin = `ALTER TABLE users ADD COLUMN is_admin INTEGER NOT NULL DEFAULT 0`
)

// User queries.
const (
	queryUpsertUser = `
INSERT INTO users (username, password, display_name, is_admin) VALUES (?, ?, ?, ?)
ON CONFLICT(username) DO UPDATE SET
    password     = excluded.password,
    display_name = excluded.display_name,
    is_admin     = excluded.is_admin`

	// queryInsertUserIfNotExists is used at startup to seed config-defined users
	// without overwriting changes made via the CLI or admin panel after first boot.
	queryInsertUserIfNotExists = `
INSERT OR IGNORE INTO users (username, password, display_name, is_admin) VALUES (?, ?, ?, ?)`

	queryGetUser = `
SELECT username, password, display_name, is_admin
FROM users
WHERE username = ?`

	queryListUsers = `
SELECT username, display_name, is_admin
FROM users
ORDER BY username ASC`

	queryDeleteUser = `
DELETE FROM users
WHERE username = ?`
)

// Channel queries.
const (
	queryUpsertChannel = `
INSERT INTO channels (code, name) VALUES (?, ?)
ON CONFLICT(code) DO UPDATE SET
    name = excluded.name`

	queryGetChannel = `
SELECT code, name
FROM channels
WHERE code = ?`

	queryListChannels = `
SELECT code, name
FROM channels
ORDER BY code ASC`

	queryDeleteChannel = `
DELETE FROM channels
WHERE code = ?`
)

// Message queries.
const (
	queryMaxMessageID = `
SELECT COALESCE(MAX(CAST(id AS INTEGER)), 0)
FROM messages`

	queryInsertMessage = `
INSERT OR IGNORE INTO messages
    (id, channel_code, channel_name, username, display_name, body, timestamp)
VALUES (?, ?, ?, ?, ?, ?, ?)`

	// queryGetMessages selects the newest `limit` rows then re-sorts oldest-first for the caller.
	queryGetMessages = `
SELECT id, channel_code, channel_name, username, display_name, body, timestamp
FROM (
    SELECT id, channel_code, channel_name, username, display_name, body, timestamp
    FROM messages
    WHERE channel_code = ?
    ORDER BY timestamp DESC
    LIMIT ?
)
ORDER BY timestamp ASC`
)
