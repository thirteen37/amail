package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS messages (
    id TEXT PRIMARY KEY,
    from_id TEXT NOT NULL,
    subject TEXT,
    body TEXT NOT NULL,
    priority TEXT DEFAULT 'normal',
    msg_type TEXT DEFAULT 'message',
    thread_id TEXT,
    reply_to_id TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (thread_id) REFERENCES messages(id),
    FOREIGN KEY (reply_to_id) REFERENCES messages(id)
);

CREATE TABLE IF NOT EXISTS recipients (
    message_id TEXT NOT NULL,
    to_id TEXT NOT NULL,
    status TEXT DEFAULT 'unread',
    read_at TIMESTAMP,
    notified_at TIMESTAMP,
    PRIMARY KEY (message_id, to_id),
    FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_inbox ON recipients(to_id, status);
CREATE INDEX IF NOT EXISTS idx_thread ON messages(thread_id);
CREATE INDEX IF NOT EXISTS idx_messages_created ON messages(created_at DESC);
`

// DB wraps the SQLite database connection
type DB struct {
	conn *sql.DB
	path string
}

// Open opens the database at the given path
func Open(path string) (*DB, error) {
	// Use connection string pragmas to ensure they apply to all pooled connections
	// - foreign_keys: enforce referential integrity
	// - journal_mode=WAL: enable concurrent read/write access
	// - busy_timeout: wait up to 5 seconds on lock contention
	connStr := path + "?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	conn, err := sql.Open("sqlite", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	return &DB{conn: conn, path: path}, nil
}

// Init initializes the database schema
func (db *DB) Init() error {
	_, err := db.conn.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to initialize schema: %w", err)
	}
	return nil
}

// Close checkpoints the WAL and closes the database connection
func (db *DB) Close() error {
	// Checkpoint WAL to minimize file size (PASSIVE doesn't block readers)
	_, _ = db.conn.Exec("PRAGMA wal_checkpoint(PASSIVE)")
	return db.conn.Close()
}

// Message represents a message in the system
type Message struct {
	ID        string
	FromID    string
	Subject   string
	Body      string
	Priority  string
	MsgType   string
	ThreadID  *string
	ReplyToID *string
	CreatedAt time.Time
}

// Recipient represents a message recipient with read status
type Recipient struct {
	MessageID  string
	ToID       string
	Status     string
	ReadAt     *time.Time
	NotifiedAt *time.Time
}

// InboxMessage combines message data with recipient-specific info
type InboxMessage struct {
	Message
	ToIDs  []string
	Status string
	ReadAt *time.Time
}

// scanInboxRows scans rows into InboxMessage slice, handling nullable fields.
// If includeStatus is true, it expects status and read_at columns in the result.
func scanInboxRows(rows *sql.Rows, includeStatus bool) ([]InboxMessage, []string, error) {
	var messages []InboxMessage
	var messageIDs []string

	for rows.Next() {
		var msg InboxMessage
		var threadID, replyToID sql.NullString
		var readAt sql.NullTime

		var err error
		if includeStatus {
			err = rows.Scan(
				&msg.ID, &msg.FromID, &msg.Subject, &msg.Body, &msg.Priority, &msg.MsgType,
				&threadID, &replyToID, &msg.CreatedAt, &msg.Status, &readAt)
		} else {
			err = rows.Scan(
				&msg.ID, &msg.FromID, &msg.Subject, &msg.Body, &msg.Priority, &msg.MsgType,
				&threadID, &replyToID, &msg.CreatedAt)
		}
		if err != nil {
			return nil, nil, fmt.Errorf("failed to scan row: %w", err)
		}

		if threadID.Valid {
			msg.ThreadID = &threadID.String
		}
		if replyToID.Valid {
			msg.ReplyToID = &replyToID.String
		}
		if readAt.Valid {
			msg.ReadAt = &readAt.Time
		}

		messages = append(messages, msg)
		messageIDs = append(messageIDs, msg.ID)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	return messages, messageIDs, nil
}

// attachRecipients fetches and assigns recipients to a slice of messages.
func (db *DB) attachRecipients(messages []InboxMessage, messageIDs []string) error {
	if len(messages) == 0 {
		return nil
	}

	recipientMap, err := db.getRecipientsForMessages(messageIDs)
	if err != nil {
		return err
	}

	for i := range messages {
		messages[i].ToIDs = recipientMap[messages[i].ID]
	}
	return nil
}

// SendMessage creates a new message and adds recipients
func (db *DB) SendMessage(msg *Message, recipients []string) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert message
	_, err = tx.Exec(`
		INSERT INTO messages (id, from_id, subject, body, priority, msg_type, thread_id, reply_to_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		msg.ID, msg.FromID, msg.Subject, msg.Body, msg.Priority, msg.MsgType, msg.ThreadID, msg.ReplyToID, msg.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert message: %w", err)
	}

	// Insert recipients
	for _, toID := range recipients {
		_, err = tx.Exec(`
			INSERT INTO recipients (message_id, to_id, status)
			VALUES (?, ?, 'unread')`,
			msg.ID, toID)
		if err != nil {
			return fmt.Errorf("failed to insert recipient %s: %w", toID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetInbox retrieves messages for a recipient
func (db *DB) GetInbox(toID string, includeRead bool) ([]InboxMessage, error) {
	query := `
		SELECT m.id, m.from_id, m.subject, m.body, m.priority, m.msg_type,
		       m.thread_id, m.reply_to_id, m.created_at, r.status, r.read_at
		FROM messages m
		JOIN recipients r ON m.id = r.message_id
		WHERE r.to_id = ?`

	if !includeRead {
		query += ` AND r.status = 'unread'`
	}

	query += ` ORDER BY m.created_at DESC`

	rows, err := db.conn.Query(query, toID)
	if err != nil {
		return nil, fmt.Errorf("failed to query inbox: %w", err)
	}
	defer rows.Close()

	messages, messageIDs, err := scanInboxRows(rows, true)
	if err != nil {
		return nil, fmt.Errorf("failed to scan inbox: %w", err)
	}

	if err := db.attachRecipients(messages, messageIDs); err != nil {
		return nil, err
	}

	return messages, nil
}

// getMessageRecipients returns all recipients for a message
func (db *DB) getMessageRecipients(messageID string) ([]string, error) {
	rows, err := db.conn.Query(`SELECT to_id FROM recipients WHERE message_id = ?`, messageID)
	if err != nil {
		return nil, fmt.Errorf("failed to query recipients: %w", err)
	}
	defer rows.Close()

	var recipients []string
	for rows.Next() {
		var toID string
		if err := rows.Scan(&toID); err != nil {
			return nil, fmt.Errorf("failed to scan recipient: %w", err)
		}
		recipients = append(recipients, toID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating recipient rows: %w", err)
	}

	return recipients, nil
}

// getRecipientsForMessages returns all recipients for multiple messages in a single query
func (db *DB) getRecipientsForMessages(messageIDs []string) (map[string][]string, error) {
	if len(messageIDs) == 0 {
		return make(map[string][]string), nil
	}

	// Build query with placeholders
	placeholders := make([]string, len(messageIDs))
	args := make([]interface{}, len(messageIDs))
	for i, id := range messageIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf(
		`SELECT message_id, to_id FROM recipients WHERE message_id IN (%s)`,
		strings.Join(placeholders, ","),
	)

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query recipients: %w", err)
	}
	defer rows.Close()

	result := make(map[string][]string)
	for rows.Next() {
		var messageID, toID string
		if err := rows.Scan(&messageID, &toID); err != nil {
			return nil, fmt.Errorf("failed to scan recipient: %w", err)
		}
		result[messageID] = append(result[messageID], toID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating recipient rows: %w", err)
	}

	return result, nil
}

// GetMessage retrieves a single message by ID
func (db *DB) GetMessage(id string) (*InboxMessage, error) {
	var msg InboxMessage
	var threadID, replyToID sql.NullString

	err := db.conn.QueryRow(`
		SELECT id, from_id, subject, body, priority, msg_type,
		       thread_id, reply_to_id, created_at
		FROM messages WHERE id = ?`, id).Scan(
		&msg.ID, &msg.FromID, &msg.Subject, &msg.Body, &msg.Priority, &msg.MsgType,
		&threadID, &replyToID, &msg.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	if threadID.Valid {
		msg.ThreadID = &threadID.String
	}
	if replyToID.Valid {
		msg.ReplyToID = &replyToID.String
	}

	// Get recipients
	toIDs, err := db.getMessageRecipients(id)
	if err != nil {
		return nil, err
	}
	msg.ToIDs = toIDs

	return &msg, nil
}

// FindMessageByPrefix finds a message by ID prefix
func (db *DB) FindMessageByPrefix(prefix string) (*InboxMessage, error) {
	var msg InboxMessage
	var threadID, replyToID sql.NullString

	// Use LIKE with prefix matching
	err := db.conn.QueryRow(`
		SELECT id, from_id, subject, body, priority, msg_type,
		       thread_id, reply_to_id, created_at
		FROM messages WHERE id LIKE ? || '%'
		LIMIT 1`, prefix).Scan(
		&msg.ID, &msg.FromID, &msg.Subject, &msg.Body, &msg.Priority, &msg.MsgType,
		&threadID, &replyToID, &msg.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find message: %w", err)
	}

	if threadID.Valid {
		msg.ThreadID = &threadID.String
	}
	if replyToID.Valid {
		msg.ReplyToID = &replyToID.String
	}

	// Get recipients
	toIDs, err := db.getMessageRecipients(msg.ID)
	if err != nil {
		return nil, err
	}
	msg.ToIDs = toIDs

	return &msg, nil
}

// GetMessageForRecipient retrieves a message with recipient-specific status
func (db *DB) GetMessageForRecipient(id, toID string) (*InboxMessage, error) {
	var msg InboxMessage
	var threadID, replyToID sql.NullString
	var readAt sql.NullTime

	err := db.conn.QueryRow(`
		SELECT m.id, m.from_id, m.subject, m.body, m.priority, m.msg_type,
		       m.thread_id, m.reply_to_id, m.created_at, r.status, r.read_at
		FROM messages m
		JOIN recipients r ON m.id = r.message_id
		WHERE m.id = ? AND r.to_id = ?`, id, toID).Scan(
		&msg.ID, &msg.FromID, &msg.Subject, &msg.Body, &msg.Priority, &msg.MsgType,
		&threadID, &replyToID, &msg.CreatedAt, &msg.Status, &readAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	if threadID.Valid {
		msg.ThreadID = &threadID.String
	}
	if replyToID.Valid {
		msg.ReplyToID = &replyToID.String
	}
	if readAt.Valid {
		msg.ReadAt = &readAt.Time
	}

	// Get all recipients
	toIDs, err := db.getMessageRecipients(id)
	if err != nil {
		return nil, err
	}
	msg.ToIDs = toIDs

	return &msg, nil
}

// MarkRead marks a message as read for a recipient
func (db *DB) MarkRead(messageID, toID string) error {
	_, err := db.conn.Exec(`
		UPDATE recipients SET status = 'read', read_at = ?
		WHERE message_id = ? AND to_id = ?`,
		time.Now(), messageID, toID)
	if err != nil {
		return fmt.Errorf("failed to mark as read: %w", err)
	}
	return nil
}

// GetUnnotified returns unread messages that haven't been notified yet
func (db *DB) GetUnnotified(toID string) ([]InboxMessage, error) {
	query := `
		SELECT m.id, m.from_id, m.subject, m.body, m.priority, m.msg_type,
		       m.thread_id, m.reply_to_id, m.created_at, r.status, r.read_at
		FROM messages m
		JOIN recipients r ON m.id = r.message_id
		WHERE r.to_id = ? AND r.status = 'unread' AND r.notified_at IS NULL
		ORDER BY m.created_at DESC`

	rows, err := db.conn.Query(query, toID)
	if err != nil {
		return nil, fmt.Errorf("failed to query unnotified: %w", err)
	}
	defer rows.Close()

	messages, messageIDs, err := scanInboxRows(rows, true)
	if err != nil {
		return nil, fmt.Errorf("failed to scan unnotified: %w", err)
	}

	if err := db.attachRecipients(messages, messageIDs); err != nil {
		return nil, err
	}

	return messages, nil
}

// MarkNotified marks a message as notified for a recipient
func (db *DB) MarkNotified(messageID, toID string) error {
	_, err := db.conn.Exec(`
		UPDATE recipients SET notified_at = ?
		WHERE message_id = ? AND to_id = ?`,
		time.Now(), messageID, toID)
	if err != nil {
		return fmt.Errorf("failed to mark as notified: %w", err)
	}
	return nil
}

// MarkAllRead marks all messages as read for a recipient
func (db *DB) MarkAllRead(toID string) (int64, error) {
	result, err := db.conn.Exec(`
		UPDATE recipients SET status = 'read', read_at = ?
		WHERE to_id = ? AND status = 'unread'`,
		time.Now(), toID)
	if err != nil {
		return 0, fmt.Errorf("failed to mark all as read: %w", err)
	}
	return result.RowsAffected()
}

// Archive marks a message as archived for a recipient
func (db *DB) Archive(messageID, toID string) error {
	_, err := db.conn.Exec(`
		UPDATE recipients SET status = 'archived'
		WHERE message_id = ? AND to_id = ?`,
		messageID, toID)
	if err != nil {
		return fmt.Errorf("failed to archive: %w", err)
	}
	return nil
}

// Delete removes a recipient from a message (soft delete for recipient)
func (db *DB) Delete(messageID, toID string) error {
	_, err := db.conn.Exec(`
		DELETE FROM recipients WHERE message_id = ? AND to_id = ?`,
		messageID, toID)
	if err != nil {
		return fmt.Errorf("failed to delete: %w", err)
	}
	return nil
}

// CountUnread returns the number of unread messages for a recipient
func (db *DB) CountUnread(toID string) (int, error) {
	var count int
	err := db.conn.QueryRow(`
		SELECT COUNT(*) FROM recipients WHERE to_id = ? AND status = 'unread'`,
		toID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count unread: %w", err)
	}
	return count, nil
}

// GetThread retrieves all messages in a thread
func (db *DB) GetThread(threadID string) ([]InboxMessage, error) {
	// Get the root message and all replies
	query := `
		SELECT m.id, m.from_id, m.subject, m.body, m.priority, m.msg_type,
		       m.thread_id, m.reply_to_id, m.created_at
		FROM messages m
		WHERE m.id = ? OR m.thread_id = ?
		ORDER BY m.created_at ASC`

	rows, err := db.conn.Query(query, threadID, threadID)
	if err != nil {
		return nil, fmt.Errorf("failed to query thread: %w", err)
	}
	defer rows.Close()

	messages, messageIDs, err := scanInboxRows(rows, false)
	if err != nil {
		return nil, fmt.Errorf("failed to scan thread: %w", err)
	}

	if err := db.attachRecipients(messages, messageIDs); err != nil {
		return nil, err
	}

	return messages, nil
}

// GetLatestUnread returns the most recent unread message for a recipient
func (db *DB) GetLatestUnread(toID string) (*InboxMessage, error) {
	messages, err := db.GetInbox(toID, false)
	if err != nil {
		return nil, err
	}
	if len(messages) == 0 {
		return nil, nil
	}
	return &messages[0], nil
}

// FindProjectRoot looks for .amail directory in current or parent directories
func FindProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		amailDir := filepath.Join(dir, ".amail")
		if info, err := os.Stat(amailDir); err == nil && info.IsDir() {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("not in an amail project (no .amail directory found)")
		}
		dir = parent
	}
}

// DBPath returns the database path for a project root
func DBPath(projectRoot string) string {
	return filepath.Join(projectRoot, ".amail", "mail.db")
}

// OpenProject opens the database for the current project
func OpenProject() (*DB, string, error) {
	root, err := FindProjectRoot()
	if err != nil {
		return nil, "", err
	}

	db, err := Open(DBPath(root))
	if err != nil {
		return nil, "", err
	}

	return db, root, nil
}
