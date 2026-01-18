package db

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestWALModeEnabled(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	var journalMode string
	err := db.conn.QueryRow("PRAGMA journal_mode").Scan(&journalMode)
	if err != nil {
		t.Fatalf("failed to query journal_mode: %v", err)
	}

	if journalMode != "wal" {
		t.Errorf("expected journal_mode 'wal', got '%s'", journalMode)
	}
}

func TestBusyTimeoutSet(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	var timeout int
	err := db.conn.QueryRow("PRAGMA busy_timeout").Scan(&timeout)
	if err != nil {
		t.Fatalf("failed to query busy_timeout: %v", err)
	}

	if timeout != 5000 {
		t.Errorf("expected busy_timeout 5000, got %d", timeout)
	}
}

func TestConcurrentSendMessages(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	const numGoroutines = 10
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			msg := &Message{
				ID:        fmt.Sprintf("msg%03d", n),
				FromID:    "sender",
				Subject:   fmt.Sprintf("Message %d", n),
				Body:      "Body",
				Priority:  "normal",
				MsgType:   "message",
				CreatedAt: time.Now(),
			}
			if err := db.SendMessage(msg, []string{"recipient"}); err != nil {
				errors <- fmt.Errorf("goroutine %d: %w", n, err)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Error(err)
	}

	// Verify all messages were persisted
	inbox, err := db.GetInbox("recipient", false)
	if err != nil {
		t.Fatalf("GetInbox failed: %v", err)
	}
	if len(inbox) != numGoroutines {
		t.Errorf("expected %d messages, got %d", numGoroutines, len(inbox))
	}
}

func TestConcurrentMarkRead(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Send one message to multiple recipients
	recipients := []string{"r1", "r2", "r3", "r4", "r5"}
	msg := &Message{
		ID:        "msg001",
		FromID:    "sender",
		Subject:   "Test",
		Body:      "Body",
		Priority:  "normal",
		MsgType:   "message",
		CreatedAt: time.Now(),
	}
	if err := db.SendMessage(msg, recipients); err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}

	// Concurrently mark read for each recipient
	var wg sync.WaitGroup
	errors := make(chan error, len(recipients))

	for _, r := range recipients {
		wg.Add(1)
		go func(toID string) {
			defer wg.Done()
			if err := db.MarkRead("msg001", toID); err != nil {
				errors <- fmt.Errorf("recipient %s: %w", toID, err)
			}
		}(r)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Error(err)
	}

	// Verify all are marked read
	for _, r := range recipients {
		count, _ := db.CountUnread(r)
		if count != 0 {
			t.Errorf("expected 0 unread for %s, got %d", r, count)
		}
	}
}

func TestReadDuringWrite(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	const numWrites = 20
	const numReads = 50

	var wg sync.WaitGroup
	writeErrors := make(chan error, numWrites)
	readErrors := make(chan error, numReads)

	// Writer goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < numWrites; i++ {
			msg := &Message{
				ID:        fmt.Sprintf("msg%03d", i),
				FromID:    "writer",
				Subject:   fmt.Sprintf("Message %d", i),
				Body:      "Body",
				Priority:  "normal",
				MsgType:   "message",
				CreatedAt: time.Now(),
			}
			if err := db.SendMessage(msg, []string{"reader"}); err != nil {
				writeErrors <- err
			}
			time.Sleep(time.Millisecond) // Small delay between writes
		}
	}()

	// Reader goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < numReads; i++ {
			if _, err := db.GetInbox("reader", false); err != nil {
				readErrors <- err
			}
			time.Sleep(time.Millisecond / 2) // Read faster than write
		}
	}()

	wg.Wait()
	close(writeErrors)
	close(readErrors)

	for err := range writeErrors {
		t.Errorf("write error: %v", err)
	}
	for err := range readErrors {
		t.Errorf("read error: %v", err)
	}
}

func TestWatchDuringAgentWrite(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	const numMessages = 10
	seenMessages := make(map[string]bool)
	var mu sync.Mutex
	done := make(chan struct{})

	// Simulate watch polling loop
	go func() {
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				messages, err := db.GetUnnotified("watcher")
				if err != nil {
					continue
				}
				for _, msg := range messages {
					mu.Lock()
					seenMessages[msg.ID] = true
					mu.Unlock()
					db.MarkNotified(msg.ID, "watcher")
				}
			}
		}
	}()

	// Agent sends messages
	for i := 0; i < numMessages; i++ {
		msg := &Message{
			ID:        fmt.Sprintf("msg%03d", i),
			FromID:    "agent",
			Subject:   fmt.Sprintf("Message %d", i),
			Body:      "Body",
			Priority:  "normal",
			MsgType:   "message",
			CreatedAt: time.Now(),
		}
		if err := db.SendMessage(msg, []string{"watcher"}); err != nil {
			t.Errorf("SendMessage failed: %v", err)
		}
		time.Sleep(5 * time.Millisecond)
	}

	// Wait for watcher to catch up
	time.Sleep(100 * time.Millisecond)
	close(done)

	// Verify all messages were seen
	mu.Lock()
	if len(seenMessages) != numMessages {
		t.Errorf("expected watcher to see %d messages, saw %d", numMessages, len(seenMessages))
	}
	mu.Unlock()
}

func TestBusyTimeoutRetry(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "amail-concurrent-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")

	// Open first connection
	db1, err := Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open db1: %v", err)
	}
	defer db1.Close()
	if err := db1.Init(); err != nil {
		t.Fatalf("failed to init db1: %v", err)
	}

	// Open second connection
	db2, err := Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open db2: %v", err)
	}
	defer db2.Close()

	// Start a transaction on db1
	tx, err := db1.conn.Begin()
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}

	// Insert via transaction
	_, err = tx.Exec(`INSERT INTO messages (id, from_id, body, created_at) VALUES (?, ?, ?, ?)`,
		"msg001", "sender", "body", time.Now())
	if err != nil {
		tx.Rollback()
		t.Fatalf("failed to insert: %v", err)
	}

	// Try to write from db2 in a goroutine (should wait)
	done := make(chan error, 1)
	go func() {
		msg := &Message{
			ID:        "msg002",
			FromID:    "sender2",
			Subject:   "Test",
			Body:      "Body",
			Priority:  "normal",
			MsgType:   "message",
			CreatedAt: time.Now(),
		}
		done <- db2.SendMessage(msg, []string{"recipient"})
	}()

	// Small delay, then commit db1's transaction
	time.Sleep(50 * time.Millisecond)
	if err := tx.Commit(); err != nil {
		t.Fatalf("failed to commit: %v", err)
	}

	// db2's write should now succeed
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("db2 write failed: %v", err)
		}
	case <-time.After(6 * time.Second):
		t.Error("db2 write timed out (busy_timeout should have handled this)")
	}
}

func TestCheckpointOnClose(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "amail-checkpoint-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	walPath := dbPath + "-wal"

	// Open and write data
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	if err := db.Init(); err != nil {
		db.Close()
		t.Fatalf("failed to init db: %v", err)
	}

	// Write several messages to grow the WAL
	for i := 0; i < 100; i++ {
		msg := &Message{
			ID:        fmt.Sprintf("msg%03d", i),
			FromID:    "sender",
			Subject:   fmt.Sprintf("Message %d", i),
			Body:      "Body content that adds some size to the WAL file",
			Priority:  "normal",
			MsgType:   "message",
			CreatedAt: time.Now(),
		}
		if err := db.SendMessage(msg, []string{"recipient"}); err != nil {
			db.Close()
			t.Fatalf("SendMessage failed: %v", err)
		}
	}

	// Check WAL exists before close
	if _, err := os.Stat(walPath); os.IsNotExist(err) {
		db.Close()
		t.Fatal("WAL file should exist before close")
	}

	// Close triggers checkpoint
	if err := db.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// After close, WAL should be small (checkpointed) or may not exist
	info, err := os.Stat(walPath)
	if err == nil {
		// WAL exists, should be small (< 4KB typically after PASSIVE checkpoint)
		// Note: exact behavior depends on SQLite version and state
		if info.Size() > 32768 { // 32KB as reasonable threshold
			t.Errorf("WAL file larger than expected after checkpoint: %d bytes", info.Size())
		}
	}
	// If WAL doesn't exist, that's also fine - it means full checkpoint happened
}
