package db

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func setupTestDB(t *testing.T) (*DB, func()) {
	t.Helper()

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "amail-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := Open(dbPath)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to open db: %v", err)
	}

	if err := db.Init(); err != nil {
		db.Close()
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to init db: %v", err)
	}

	cleanup := func() {
		db.Close()
		os.RemoveAll(tmpDir)
	}

	return db, cleanup
}

func TestOpenAndInit(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	if db == nil {
		t.Fatal("expected non-nil db")
	}
}

func TestSendMessage(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	msg := &Message{
		ID:        "msg001",
		FromID:    "pm",
		Subject:   "Test Subject",
		Body:      "Test body content",
		Priority:  "normal",
		MsgType:   "message",
		CreatedAt: time.Now(),
	}

	err := db.SendMessage(msg, []string{"dev", "qa"})
	if err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}

	// Verify message was created
	retrieved, err := db.GetMessage("msg001")
	if err != nil {
		t.Fatalf("GetMessage failed: %v", err)
	}
	if retrieved == nil {
		t.Fatal("expected message to exist")
	}
	if retrieved.FromID != "pm" {
		t.Errorf("expected FromID 'pm', got '%s'", retrieved.FromID)
	}
	if retrieved.Subject != "Test Subject" {
		t.Errorf("expected Subject 'Test Subject', got '%s'", retrieved.Subject)
	}
	if len(retrieved.ToIDs) != 2 {
		t.Errorf("expected 2 recipients, got %d", len(retrieved.ToIDs))
	}
}

func TestGetInbox(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Send two messages to dev
	msg1 := &Message{
		ID:        "msg001",
		FromID:    "pm",
		Subject:   "First",
		Body:      "First message",
		Priority:  "normal",
		MsgType:   "message",
		CreatedAt: time.Now().Add(-time.Hour),
	}
	db.SendMessage(msg1, []string{"dev"})

	msg2 := &Message{
		ID:        "msg002",
		FromID:    "qa",
		Subject:   "Second",
		Body:      "Second message",
		Priority:  "high",
		MsgType:   "request",
		CreatedAt: time.Now(),
	}
	db.SendMessage(msg2, []string{"dev"})

	// Get dev's inbox (unread only)
	inbox, err := db.GetInbox("dev", false)
	if err != nil {
		t.Fatalf("GetInbox failed: %v", err)
	}
	if len(inbox) != 2 {
		t.Errorf("expected 2 messages, got %d", len(inbox))
	}

	// Most recent should be first
	if inbox[0].ID != "msg002" {
		t.Errorf("expected most recent message first, got %s", inbox[0].ID)
	}

	// pm's inbox should be empty
	pmInbox, err := db.GetInbox("pm", false)
	if err != nil {
		t.Fatalf("GetInbox for pm failed: %v", err)
	}
	if len(pmInbox) != 0 {
		t.Errorf("expected 0 messages for pm, got %d", len(pmInbox))
	}
}

func TestMarkRead(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	msg := &Message{
		ID:        "msg001",
		FromID:    "pm",
		Subject:   "Test",
		Body:      "Body",
		Priority:  "normal",
		MsgType:   "message",
		CreatedAt: time.Now(),
	}
	db.SendMessage(msg, []string{"dev"})

	// Initially unread
	count, _ := db.CountUnread("dev")
	if count != 1 {
		t.Errorf("expected 1 unread, got %d", count)
	}

	// Mark as read
	err := db.MarkRead("msg001", "dev")
	if err != nil {
		t.Fatalf("MarkRead failed: %v", err)
	}

	// Now should be 0 unread
	count, _ = db.CountUnread("dev")
	if count != 0 {
		t.Errorf("expected 0 unread after marking read, got %d", count)
	}

	// Should still appear in inbox with includeRead=true
	inbox, _ := db.GetInbox("dev", true)
	if len(inbox) != 1 {
		t.Errorf("expected 1 message with includeRead, got %d", len(inbox))
	}
}

func TestMarkAllRead(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Send 3 messages
	for i := 1; i <= 3; i++ {
		msg := &Message{
			ID:        "msg00" + string(rune('0'+i)),
			FromID:    "pm",
			Subject:   "Test",
			Body:      "Body",
			Priority:  "normal",
			MsgType:   "message",
			CreatedAt: time.Now(),
		}
		db.SendMessage(msg, []string{"dev"})
	}

	count, _ := db.CountUnread("dev")
	if count != 3 {
		t.Errorf("expected 3 unread, got %d", count)
	}

	affected, err := db.MarkAllRead("dev")
	if err != nil {
		t.Fatalf("MarkAllRead failed: %v", err)
	}
	if affected != 3 {
		t.Errorf("expected 3 affected, got %d", affected)
	}

	count, _ = db.CountUnread("dev")
	if count != 0 {
		t.Errorf("expected 0 unread after MarkAllRead, got %d", count)
	}
}

func TestArchive(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	msg := &Message{
		ID:        "msg001",
		FromID:    "pm",
		Subject:   "Test",
		Body:      "Body",
		Priority:  "normal",
		MsgType:   "message",
		CreatedAt: time.Now(),
	}
	db.SendMessage(msg, []string{"dev"})

	err := db.Archive("msg001", "dev")
	if err != nil {
		t.Fatalf("Archive failed: %v", err)
	}

	// Should not appear in unread
	inbox, _ := db.GetInbox("dev", false)
	if len(inbox) != 0 {
		t.Errorf("expected 0 unread after archive, got %d", len(inbox))
	}
}

func TestDelete(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	msg := &Message{
		ID:        "msg001",
		FromID:    "pm",
		Subject:   "Test",
		Body:      "Body",
		Priority:  "normal",
		MsgType:   "message",
		CreatedAt: time.Now(),
	}
	db.SendMessage(msg, []string{"dev", "qa"})

	// Delete for dev only
	err := db.Delete("msg001", "dev")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// dev should not see it
	devInbox, _ := db.GetInbox("dev", true)
	if len(devInbox) != 0 {
		t.Errorf("expected 0 messages for dev after delete, got %d", len(devInbox))
	}

	// qa should still see it
	qaInbox, _ := db.GetInbox("qa", true)
	if len(qaInbox) != 1 {
		t.Errorf("expected 1 message for qa, got %d", len(qaInbox))
	}
}

func TestThreading(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Original message
	msg1 := &Message{
		ID:        "msg001",
		FromID:    "pm",
		Subject:   "Original",
		Body:      "Original message",
		Priority:  "normal",
		MsgType:   "message",
		CreatedAt: time.Now().Add(-2 * time.Hour),
	}
	db.SendMessage(msg1, []string{"dev"})

	// Reply (part of thread)
	threadID := "msg001"
	msg2 := &Message{
		ID:        "msg002",
		FromID:    "dev",
		Subject:   "RE: Original",
		Body:      "Reply message",
		Priority:  "normal",
		MsgType:   "response",
		ThreadID:  &threadID,
		ReplyToID: &threadID,
		CreatedAt: time.Now().Add(-time.Hour),
	}
	db.SendMessage(msg2, []string{"pm"})

	// Another reply
	msg3 := &Message{
		ID:        "msg003",
		FromID:    "pm",
		Subject:   "RE: Original",
		Body:      "Second reply",
		Priority:  "normal",
		MsgType:   "response",
		ThreadID:  &threadID,
		ReplyToID: &msg2.ID,
		CreatedAt: time.Now(),
	}
	db.SendMessage(msg3, []string{"dev"})

	// Get thread
	thread, err := db.GetThread("msg001")
	if err != nil {
		t.Fatalf("GetThread failed: %v", err)
	}
	if len(thread) != 3 {
		t.Errorf("expected 3 messages in thread, got %d", len(thread))
	}

	// Should be in chronological order
	if thread[0].ID != "msg001" {
		t.Errorf("expected first message to be msg001, got %s", thread[0].ID)
	}
	if thread[2].ID != "msg003" {
		t.Errorf("expected last message to be msg003, got %s", thread[2].ID)
	}
}

func TestFindMessageByPrefix(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	msg := &Message{
		ID:        "abc123def456",
		FromID:    "pm",
		Subject:   "Test",
		Body:      "Body",
		Priority:  "normal",
		MsgType:   "message",
		CreatedAt: time.Now(),
	}
	db.SendMessage(msg, []string{"dev"})

	// Find by prefix
	found, err := db.FindMessageByPrefix("abc123")
	if err != nil {
		t.Fatalf("FindMessageByPrefix failed: %v", err)
	}
	if found == nil {
		t.Fatal("expected to find message by prefix")
	}
	if found.ID != "abc123def456" {
		t.Errorf("expected ID 'abc123def456', got '%s'", found.ID)
	}

	// Short prefix
	found, _ = db.FindMessageByPrefix("abc")
	if found == nil {
		t.Fatal("expected to find message by short prefix")
	}

	// Non-existent prefix
	found, _ = db.FindMessageByPrefix("xyz")
	if found != nil {
		t.Error("expected nil for non-existent prefix")
	}
}

func TestGetLatestUnread(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Send messages with different times
	msg1 := &Message{
		ID:        "msg001",
		FromID:    "pm",
		Subject:   "Older",
		Body:      "Body",
		Priority:  "normal",
		MsgType:   "message",
		CreatedAt: time.Now().Add(-time.Hour),
	}
	db.SendMessage(msg1, []string{"dev"})

	msg2 := &Message{
		ID:        "msg002",
		FromID:    "qa",
		Subject:   "Newer",
		Body:      "Body",
		Priority:  "normal",
		MsgType:   "message",
		CreatedAt: time.Now(),
	}
	db.SendMessage(msg2, []string{"dev"})

	latest, err := db.GetLatestUnread("dev")
	if err != nil {
		t.Fatalf("GetLatestUnread failed: %v", err)
	}
	if latest == nil {
		t.Fatal("expected latest message")
	}
	if latest.ID != "msg002" {
		t.Errorf("expected latest to be msg002, got %s", latest.ID)
	}
}

func TestMultipleRecipients(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	msg := &Message{
		ID:        "msg001",
		FromID:    "pm",
		Subject:   "Broadcast",
		Body:      "To everyone",
		Priority:  "normal",
		MsgType:   "message",
		CreatedAt: time.Now(),
	}
	db.SendMessage(msg, []string{"dev", "qa", "user"})

	// Each recipient should have the message
	for _, role := range []string{"dev", "qa", "user"} {
		inbox, _ := db.GetInbox(role, false)
		if len(inbox) != 1 {
			t.Errorf("expected 1 message for %s, got %d", role, len(inbox))
		}
	}

	// pm (sender) should not have it
	pmInbox, _ := db.GetInbox("pm", false)
	if len(pmInbox) != 0 {
		t.Errorf("expected 0 messages for pm, got %d", len(pmInbox))
	}

	// ToIDs should include all recipients
	retrieved, _ := db.GetMessage("msg001")
	if len(retrieved.ToIDs) != 3 {
		t.Errorf("expected 3 ToIDs, got %d", len(retrieved.ToIDs))
	}
}
