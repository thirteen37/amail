package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/thirteen37/amail/internal/config"
	"github.com/thirteen37/amail/internal/db"
)

func setupTestDB(t *testing.T) (*db.DB, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "amail-tui-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tmpDir, "test.db")
	database, err := db.Open(dbPath)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to open db: %v", err)
	}

	if err := database.Init(); err != nil {
		database.Close()
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to init db: %v", err)
	}

	cleanup := func() {
		database.Close()
		os.RemoveAll(tmpDir)
	}

	return database, cleanup
}

func testConfig() *config.Config {
	return &config.Config{
		Agents: config.AgentsConfig{
			Roles: []string{"dev", "pm", "qa"},
		},
		Groups:   make(map[string][]string),
		Identity: config.IdentityConfig{Tmux: make(map[string]string)},
		Watch:    config.WatchConfig{Interval: 2},
		Notify:   make(map[string]config.NotifyConfig),
	}
}

func TestNewModel(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	cfg := testConfig()
	m := NewModel(database, cfg, "dev")

	if m.identity != "dev" {
		t.Errorf("identity = %q, want %q", m.identity, "dev")
	}

	if m.view != ViewInbox {
		t.Errorf("initial view = %v, want ViewInbox", m.view)
	}

	// AllRoles returns configured roles + reserved "user" role
	if len(m.mailboxes) != 4 {
		t.Errorf("mailboxes count = %d, want 4 (3 roles + user)", len(m.mailboxes))
	}
}

func TestViewInboxRender(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	cfg := testConfig()
	m := NewModel(database, cfg, "dev")
	m.view = ViewInbox

	view := m.View()

	if !strings.Contains(view, "amail") {
		t.Error("inbox view should contain 'amail' title")
	}
	if !strings.Contains(view, "dev") {
		t.Error("inbox view should contain identity 'dev'")
	}
	if !strings.Contains(view, "compose") {
		t.Error("inbox view should contain help text with 'compose'")
	}
}

func TestViewComposeRender(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	cfg := testConfig()
	m := NewModel(database, cfg, "dev")
	m.view = ViewCompose

	view := m.View()

	if !strings.Contains(view, "Compose") {
		t.Error("compose view should contain 'Compose' title")
	}
	if !strings.Contains(view, "To:") {
		t.Error("compose view should contain 'To:' field")
	}
	if !strings.Contains(view, "Subject:") {
		t.Error("compose view should contain 'Subject:' field")
	}
}

func TestViewMessageRender(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	cfg := testConfig()
	m := NewModel(database, cfg, "dev")
	m.view = ViewMessage
	m.currentMessage = &db.InboxMessage{
		Message: db.Message{
			ID:        "test123",
			FromID:    "pm",
			Subject:   "Test Subject",
			Body:      "Test body content",
			Priority:  "normal",
			MsgType:   "message",
			CreatedAt: time.Now(),
		},
		ToIDs:  []string{"dev"},
		Status: "unread",
	}
	m.messageView.SetContent(m.formatMessage(m.currentMessage))

	view := m.View()

	if !strings.Contains(view, "Test Subject") {
		t.Error("message view should contain subject")
	}
}

func TestViewMessageNoMessage(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	cfg := testConfig()
	m := NewModel(database, cfg, "dev")
	m.view = ViewMessage
	m.currentMessage = nil

	view := m.View()

	if !strings.Contains(view, "No message selected") {
		t.Error("message view without message should show 'No message selected'")
	}
}

func TestWindowSizeUpdate(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	cfg := testConfig()
	m := NewModel(database, cfg, "dev")

	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if updated.width != 120 {
		t.Errorf("width = %d, want 120", updated.width)
	}
	if updated.height != 40 {
		t.Errorf("height = %d, want 40", updated.height)
	}
}

func TestKeyQuit(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	cfg := testConfig()
	m := NewModel(database, cfg, "dev")
	m.view = ViewInbox

	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	_, cmd := m.Update(msg)

	if cmd == nil {
		t.Error("ctrl+c should return a quit command")
	}
}

func TestKeyCompose(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	cfg := testConfig()
	m := NewModel(database, cfg, "dev")
	m.view = ViewInbox

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}}
	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if updated.view != ViewCompose {
		t.Errorf("pressing 'c' should switch to compose view, got %v", updated.view)
	}
}

func TestKeyBackFromCompose(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	cfg := testConfig()
	m := NewModel(database, cfg, "dev")
	m.view = ViewCompose

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if updated.view != ViewInbox {
		t.Errorf("pressing esc in compose should return to inbox, got %v", updated.view)
	}
}

func TestKeyBackFromMessage(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	cfg := testConfig()
	m := NewModel(database, cfg, "dev")
	m.view = ViewMessage

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if updated.view != ViewInbox {
		t.Errorf("pressing esc in message view should return to inbox, got %v", updated.view)
	}
}

func TestTabSwitchesMailbox(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	cfg := testConfig()
	m := NewModel(database, cfg, "dev")
	m.view = ViewInbox
	m.selectedMailbox = 0

	msg := tea.KeyMsg{Type: tea.KeyTab}
	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if updated.selectedMailbox != 1 {
		t.Errorf("tab should increment selectedMailbox, got %d", updated.selectedMailbox)
	}
}

func TestFormatMessage(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	cfg := testConfig()
	m := NewModel(database, cfg, "dev")

	msg := &db.InboxMessage{
		Message: db.Message{
			ID:        "test123",
			FromID:    "pm",
			Subject:   "Test Subject",
			Body:      "Hello, this is the body.",
			Priority:  "high",
			MsgType:   "message",
			CreatedAt: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
		},
		ToIDs:  []string{"dev", "qa"},
		Status: "unread",
	}

	formatted := m.formatMessage(msg)

	if !strings.Contains(formatted, "pm") {
		t.Error("formatted message should contain sender")
	}
	if !strings.Contains(formatted, "dev") {
		t.Error("formatted message should contain recipient")
	}
	if !strings.Contains(formatted, "Test Subject") {
		t.Error("formatted message should contain subject")
	}
	if !strings.Contains(formatted, "Hello, this is the body.") {
		t.Error("formatted message should contain body")
	}
	if !strings.Contains(formatted, "high") {
		t.Error("formatted message should contain priority")
	}
}

func TestInboxMsgUpdate(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	cfg := testConfig()
	m := NewModel(database, cfg, "dev")

	messages := []db.InboxMessage{
		{
			Message: db.Message{
				ID:        "msg1",
				FromID:    "pm",
				Subject:   "First",
				Body:      "Body 1",
				Priority:  "normal",
				CreatedAt: time.Now(),
			},
			ToIDs:  []string{"dev"},
			Status: "unread",
		},
	}

	msg := inboxMsg{messages: messages, err: nil}
	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if len(updated.messages) != 1 {
		t.Errorf("messages count = %d, want 1", len(updated.messages))
	}
}

func TestStatusMsgUpdate(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	cfg := testConfig()
	m := NewModel(database, cfg, "dev")

	msg := statusMsg("Message sent!")
	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if updated.statusMsg != "Message sent!" {
		t.Errorf("statusMsg = %q, want %q", updated.statusMsg, "Message sent!")
	}
}

func TestErrMsgUpdate(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	cfg := testConfig()
	m := NewModel(database, cfg, "dev")

	testErr := errMsg{err: os.ErrNotExist}
	newModel, _ := m.Update(testErr)
	updated := newModel.(Model)

	if updated.err != os.ErrNotExist {
		t.Errorf("err = %v, want %v", updated.err, os.ErrNotExist)
	}
}

func TestUpdateInboxTable(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	cfg := testConfig()
	m := NewModel(database, cfg, "dev")

	m.messages = []db.InboxMessage{
		{
			Message: db.Message{
				ID:        "verylongmessageid123",
				FromID:    "pm",
				Subject:   "This is a very long subject that should be truncated",
				Body:      "Body",
				Priority:  "urgent",
				CreatedAt: time.Now(),
			},
			ToIDs:  []string{"dev"},
			Status: "unread",
		},
		{
			Message: db.Message{
				ID:        "msg2",
				FromID:    "qa",
				Subject:   "Short",
				Body:      "Body 2",
				Priority:  "high",
				CreatedAt: time.Now().Add(-2 * time.Hour),
			},
			ToIDs:  []string{"dev"},
			Status: "read",
		},
	}

	m.updateInboxTable()

	// Table should have 2 rows after update
	// We can't easily inspect the rows directly, but the method shouldn't panic
}

func TestViewMailboxesRender(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	cfg := testConfig()
	m := NewModel(database, cfg, "dev")
	m.view = ViewMailboxes
	m.selectedMailbox = 1

	view := m.View()

	if !strings.Contains(view, "Mailboxes") {
		t.Error("mailboxes view should contain 'Mailboxes' title")
	}
}

func TestInit(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	cfg := testConfig()
	m := NewModel(database, cfg, "dev")

	cmd := m.Init()
	if cmd == nil {
		t.Error("Init should return a command to refresh inbox")
	}
}
