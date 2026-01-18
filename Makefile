.PHONY: build install clean test coverage install-skill

VERSION := 0.2.0
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -ldflags "-X github.com/thirteen37/amail/internal/cli.Version=$(VERSION) \
                     -X github.com/thirteen37/amail/internal/cli.GitCommit=$(GIT_COMMIT) \
                     -X github.com/thirteen37/amail/internal/cli.BuildDate=$(BUILD_DATE)"

build:
	go build $(LDFLAGS) -o amail ./cmd/amail

install:
	go install $(LDFLAGS) ./cmd/amail

clean:
	rm -f amail
	rm -rf /tmp/claude/amail-test

test:
	go test ./...

coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out
	@rm -f coverage.out

install-skill:
	@mkdir -p ~/.claude/skills/amail
	@cp skills/amail/SKILL.md ~/.claude/skills/amail/SKILL.md
	@echo "✓ Installed skill to ~/.claude/skills/amail/"
	@echo "  Restart Claude Code to use the skill"

# Run a quick demo
demo: build
	@echo "Creating demo project..."
	@rm -rf /tmp/claude/amail-demo
	@mkdir -p /tmp/claude/amail-demo
	@cd /tmp/claude/amail-demo && $(CURDIR)/amail init --agents pm,dev,qa
	@echo ""
	@echo "Sending test messages..."
	@cd /tmp/claude/amail-demo && AMAIL_IDENTITY=pm $(CURDIR)/amail send dev "Feature request" "Please implement user authentication"
	@cd /tmp/claude/amail-demo && AMAIL_IDENTITY=dev $(CURDIR)/amail send pm,qa "Implementation done" "Auth feature is ready for review"
	@echo ""
	@echo "Checking dev's inbox:"
	@cd /tmp/claude/amail-demo && AMAIL_IDENTITY=dev $(CURDIR)/amail inbox
	@echo ""
	@echo "Stats:"
	@cd /tmp/claude/amail-demo && $(CURDIR)/amail stats
