package notify

import (
	"testing"
	"time"

	"github.com/thirteen37/amail/internal/db"
)

func TestFromInboxMessage(t *testing.T) {
	inboxMsg := &db.InboxMessage{
		Message: db.Message{
			ID:        "msg001",
			FromID:    "pm",
			Subject:   "Test Subject",
			Body:      "Test body",
			Priority:  "high",
			MsgType:   "request",
			CreatedAt: time.Now(),
		},
		ToIDs: []string{"dev", "qa"},
	}

	msg := FromInboxMessage(inboxMsg)

	if msg.ID != "msg001" {
		t.Errorf("expected ID 'msg001', got '%s'", msg.ID)
	}
	if msg.From != "pm" {
		t.Errorf("expected From 'pm', got '%s'", msg.From)
	}
	if msg.To != "dev,qa" {
		t.Errorf("expected To 'dev,qa', got '%s'", msg.To)
	}
	if msg.Subject != "Test Subject" {
		t.Errorf("expected Subject 'Test Subject', got '%s'", msg.Subject)
	}
	if msg.Priority != "high" {
		t.Errorf("expected Priority 'high', got '%s'", msg.Priority)
	}
}

func TestSubstituteTemplateVars(t *testing.T) {
	// substituteTemplateVars now converts placeholders to shell variable references
	tests := []struct {
		template string
		expected string
	}{
		{"echo {from}", `echo "$AMAIL_FROM"`},
		{"echo {subject}", `echo "$AMAIL_SUBJECT"`},
		{"{from} -> {to}", `"$AMAIL_FROM" -> "$AMAIL_TO"`},
		{"[{priority}] {subject}", `["$AMAIL_PRIORITY"] "$AMAIL_SUBJECT"`},
		{"ID: {id}", `ID: "$AMAIL_ID"`},
		{"{type}: {body}", `"$AMAIL_TYPE": "$AMAIL_BODY"`},
		{"Time: {timestamp}", `Time: "$AMAIL_TIMESTAMP"`},
		{"No vars here", "No vars here"},
	}

	for _, tt := range tests {
		t.Run(tt.template, func(t *testing.T) {
			result := substituteTemplateVars(tt.template)
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestTruncateForNotification(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 100, "short"},
		{"exactly ten", 11, "exactly ten"},
		{"this is a longer string", 10, "this is..."},
		{"with\nnewlines\nhere", 100, "with newlines here"},
		{"", 10, ""},
		// UTF-8 handling - truncate by runes, not bytes
		{"日本語テスト", 5, "日本..."},
		{"Hello世界", 8, "Hello世界"},
		{"🚀🎉🎊🎁", 3, "🚀🎉🎊"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := truncateForNotification(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestExecute(t *testing.T) {
	msg := &Message{
		From:    "pm",
		Subject: "Test",
	}

	// Simple command that should succeed
	err := Execute("true", msg)
	if err != nil {
		t.Errorf("expected success, got error: %v", err)
	}

	// Command with template vars - now uses environment variables
	// {from} becomes "$AMAIL_FROM" which expands to "pm" from env
	err = Execute("test {from} = 'pm'", msg)
	if err != nil {
		t.Errorf("expected success with template, got error: %v", err)
	}
}

func TestExecuteAll(t *testing.T) {
	msg := &Message{
		From:    "pm",
		Subject: "Test",
	}

	// Mix of successful and failing commands
	commands := []string{
		"true",
		"false", // This will fail
		"true",
	}

	errors := ExecuteAll(commands, msg)

	// Should have one error (from 'false')
	if len(errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(errors))
	}
}

func TestExecuteAllSuccess(t *testing.T) {
	msg := &Message{
		From:    "pm",
		Subject: "Test",
	}

	commands := []string{
		"true",
		"true",
	}

	errors := ExecuteAll(commands, msg)

	if len(errors) != 0 {
		t.Errorf("expected 0 errors, got %d", len(errors))
	}
}
