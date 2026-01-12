package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/thirteen37/amail/internal/config"
	"github.com/thirteen37/amail/internal/db"
)

// View represents the current view mode
type View int

const (
	ViewInbox View = iota
	ViewMessage
	ViewCompose
	ViewMailboxes
)

// Model is the main TUI model
type Model struct {
	db       *db.DB
	cfg      *config.Config
	identity string

	// Current view
	view View

	// Components
	inboxTable    table.Model
	messageView   viewport.Model
	composeInputs []textinput.Model
	composeBody   textarea.Model

	// Data
	messages       []db.InboxMessage
	currentMessage *db.InboxMessage
	mailboxes      []string
	selectedMailbox int

	// Compose state
	composeTo      string
	composeSubject string

	// Dimensions
	width  int
	height int

	// Status
	statusMsg string
	err       error
}

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			MarginBottom(1)

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("57")).
			Bold(true)

	unreadStyle = lipgloss.NewStyle().
			Bold(true)

	priorityUrgentStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("196"))

	priorityHighStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("208"))

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39"))

	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62"))
)

// Key bindings
type keyMap struct {
	Up       key.Binding
	Down     key.Binding
	Enter    key.Binding
	Back     key.Binding
	Compose  key.Binding
	Reply    key.Binding
	ReplyAll key.Binding
	Delete   key.Binding
	MarkRead key.Binding
	Refresh  key.Binding
	Tab      key.Binding
	Quit     key.Binding
	Send     key.Binding
	Cancel   key.Binding
}

var keys = keyMap{
	Up:       key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
	Down:     key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
	Enter:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
	Back:     key.NewBinding(key.WithKeys("esc", "q"), key.WithHelp("esc/q", "back")),
	Compose:  key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "compose")),
	Reply:    key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reply")),
	ReplyAll: key.NewBinding(key.WithKeys("R"), key.WithHelp("R", "reply all")),
	Delete:   key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
	MarkRead: key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "mark read")),
	Refresh:  key.NewBinding(key.WithKeys("g"), key.WithHelp("g", "refresh")),
	Tab:      key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "switch mailbox")),
	Quit:     key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "quit")),
	Send:     key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("ctrl+s", "send")),
	Cancel:   key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
}

// NewModel creates a new TUI model
func NewModel(database *db.DB, cfg *config.Config, identity string) Model {
	// Create inbox table
	columns := []table.Column{
		{Title: "", Width: 1},
		{Title: "ID", Width: 8},
		{Title: "From", Width: 12},
		{Title: "Subject", Width: 30},
		{Title: "Priority", Width: 8},
		{Title: "Time", Width: 12},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true)
	s.Selected = selectedStyle
	t.SetStyles(s)

	// Create viewport for message view
	vp := viewport.New(80, 20)
	vp.Style = borderStyle

	// Create compose inputs
	toInput := textinput.New()
	toInput.Placeholder = "recipient"
	toInput.CharLimit = 100

	subjectInput := textinput.New()
	subjectInput.Placeholder = "subject"
	subjectInput.CharLimit = 200

	bodyInput := textarea.New()
	bodyInput.Placeholder = "Message body..."
	bodyInput.CharLimit = 10000

	// Get all mailboxes
	mailboxes := cfg.AllRoles()

	return Model{
		db:            database,
		cfg:           cfg,
		identity:      identity,
		view:          ViewInbox,
		inboxTable:    t,
		messageView:   vp,
		composeInputs: []textinput.Model{toInput, subjectInput},
		composeBody:   bodyInput,
		mailboxes:     mailboxes,
		width:         80,
		height:        24,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return m.refreshInbox()
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.inboxTable.SetHeight(msg.Height - 8)
		m.inboxTable.SetWidth(msg.Width - 4)
		m.messageView.Width = msg.Width - 4
		m.messageView.Height = msg.Height - 10
		return m, nil

	case tea.KeyMsg:
		switch m.view {
		case ViewInbox:
			return m.updateInbox(msg)
		case ViewMessage:
			return m.updateMessage(msg)
		case ViewCompose:
			return m.updateCompose(msg)
		case ViewMailboxes:
			return m.updateMailboxes(msg)
		}

	case inboxMsg:
		m.messages = msg.messages
		m.err = msg.err
		m.updateInboxTable()
		return m, nil

	case statusMsg:
		m.statusMsg = string(msg)
		return m, nil

	case errMsg:
		m.err = msg.err
		return m, nil
	}

	// Update focused component
	switch m.view {
	case ViewInbox:
		m.inboxTable, cmd = m.inboxTable.Update(msg)
		cmds = append(cmds, cmd)
	case ViewMessage:
		m.messageView, cmd = m.messageView.Update(msg)
		cmds = append(cmds, cmd)
	case ViewCompose:
		// Update active input
		for i := range m.composeInputs {
			m.composeInputs[i], cmd = m.composeInputs[i].Update(msg)
			cmds = append(cmds, cmd)
		}
		m.composeBody, cmd = m.composeBody.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) updateInbox(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, keys.Enter):
		if len(m.messages) > 0 {
			idx := m.inboxTable.Cursor()
			if idx < len(m.messages) {
				m.currentMessage = &m.messages[idx]
				m.view = ViewMessage
				m.messageView.SetContent(m.formatMessage(m.currentMessage))
				m.messageView.GotoTop()
				// Mark as read
				m.db.MarkRead(m.currentMessage.ID, m.identity)
			}
		}
		return m, nil

	case key.Matches(msg, keys.Compose):
		m.view = ViewCompose
		m.composeInputs[0].SetValue("")
		m.composeInputs[1].SetValue("")
		m.composeBody.SetValue("")
		m.composeInputs[0].Focus()
		return m, nil

	case key.Matches(msg, keys.Delete):
		if len(m.messages) > 0 {
			idx := m.inboxTable.Cursor()
			if idx < len(m.messages) {
				msg := m.messages[idx]
				m.db.Delete(msg.ID, m.identity)
				return m, m.refreshInbox()
			}
		}
		return m, nil

	case key.Matches(msg, keys.MarkRead):
		if len(m.messages) > 0 {
			idx := m.inboxTable.Cursor()
			if idx < len(m.messages) {
				msg := m.messages[idx]
				m.db.MarkRead(msg.ID, m.identity)
				return m, m.refreshInbox()
			}
		}
		return m, nil

	case key.Matches(msg, keys.Refresh):
		return m, m.refreshInbox()

	case key.Matches(msg, keys.Tab):
		m.selectedMailbox = (m.selectedMailbox + 1) % len(m.mailboxes)
		m.identity = m.mailboxes[m.selectedMailbox]
		return m, m.refreshInbox()
	}

	var cmd tea.Cmd
	m.inboxTable, cmd = m.inboxTable.Update(msg)
	return m, cmd
}

func (m Model) updateMessage(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Back):
		m.view = ViewInbox
		return m, m.refreshInbox()

	case key.Matches(msg, keys.Reply):
		if m.currentMessage != nil {
			m.view = ViewCompose
			m.composeInputs[0].SetValue(m.currentMessage.FromID)
			m.composeInputs[1].SetValue("RE: " + m.currentMessage.Subject)
			m.composeBody.SetValue("")
			m.composeBody.Focus()
		}
		return m, nil

	case key.Matches(msg, keys.ReplyAll):
		if m.currentMessage != nil {
			m.view = ViewCompose
			recipients := []string{m.currentMessage.FromID}
			for _, to := range m.currentMessage.ToIDs {
				if to != m.identity {
					recipients = append(recipients, to)
				}
			}
			m.composeInputs[0].SetValue(strings.Join(recipients, ","))
			m.composeInputs[1].SetValue("RE: " + m.currentMessage.Subject)
			m.composeBody.SetValue("")
			m.composeBody.Focus()
		}
		return m, nil

	case key.Matches(msg, keys.Quit):
		return m, tea.Quit
	}

	var cmd tea.Cmd
	m.messageView, cmd = m.messageView.Update(msg)
	return m, cmd
}

func (m Model) updateCompose(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Cancel):
		m.view = ViewInbox
		return m, nil

	case key.Matches(msg, keys.Send):
		to := m.composeInputs[0].Value()
		subject := m.composeInputs[1].Value()
		body := m.composeBody.Value()

		if to == "" || body == "" {
			m.statusMsg = "To and body are required"
			return m, nil
		}

		return m, m.sendMessage(to, subject, body)

	case msg.String() == "tab":
		// Cycle through inputs
		for i := range m.composeInputs {
			if m.composeInputs[i].Focused() {
				m.composeInputs[i].Blur()
				if i+1 < len(m.composeInputs) {
					m.composeInputs[i+1].Focus()
				} else {
					m.composeBody.Focus()
				}
				return m, nil
			}
		}
		if m.composeBody.Focused() {
			m.composeBody.Blur()
			m.composeInputs[0].Focus()
		}
		return m, nil
	}

	var cmd tea.Cmd
	var cmds []tea.Cmd

	for i := range m.composeInputs {
		m.composeInputs[i], cmd = m.composeInputs[i].Update(msg)
		cmds = append(cmds, cmd)
	}
	m.composeBody, cmd = m.composeBody.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m Model) updateMailboxes(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Back):
		m.view = ViewInbox
		return m, nil
	case key.Matches(msg, keys.Quit):
		return m, tea.Quit
	}
	return m, nil
}

// View renders the UI
func (m Model) View() string {
	var content string

	switch m.view {
	case ViewInbox:
		content = m.viewInbox()
	case ViewMessage:
		content = m.viewMessage()
	case ViewCompose:
		content = m.viewCompose()
	case ViewMailboxes:
		content = m.viewMailboxes()
	}

	return content
}

func (m Model) viewInbox() string {
	var b strings.Builder

	// Title with mailbox selector
	title := fmt.Sprintf("📬 amail - %s", m.identity)
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n")

	// Table
	b.WriteString(m.inboxTable.View())
	b.WriteString("\n")

	// Status
	if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
	} else if m.statusMsg != "" {
		b.WriteString(statusStyle.Render(m.statusMsg))
	}
	b.WriteString("\n")

	// Help
	help := "↑/↓: navigate • enter: read • c: compose • r: reply • d: delete • m: mark read • g: refresh • tab: switch mailbox • q: quit"
	b.WriteString(helpStyle.Render(help))

	return b.String()
}

func (m Model) viewMessage() string {
	var b strings.Builder

	if m.currentMessage == nil {
		return "No message selected"
	}

	title := fmt.Sprintf("📧 %s", m.currentMessage.Subject)
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n")

	b.WriteString(m.messageView.View())
	b.WriteString("\n")

	help := "↑/↓: scroll • r: reply • R: reply all • esc/q: back"
	b.WriteString(helpStyle.Render(help))

	return b.String()
}

func (m Model) viewCompose() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("✏️  Compose Message"))
	b.WriteString("\n\n")

	b.WriteString(headerStyle.Render("To: "))
	b.WriteString(m.composeInputs[0].View())
	b.WriteString("\n")

	b.WriteString(headerStyle.Render("Subject: "))
	b.WriteString(m.composeInputs[1].View())
	b.WriteString("\n\n")

	b.WriteString(headerStyle.Render("Message:"))
	b.WriteString("\n")
	b.WriteString(m.composeBody.View())
	b.WriteString("\n\n")

	if m.statusMsg != "" {
		b.WriteString(statusStyle.Render(m.statusMsg))
		b.WriteString("\n")
	}

	help := "tab: next field • ctrl+s: send • esc: cancel"
	b.WriteString(helpStyle.Render(help))

	return b.String()
}

func (m Model) viewMailboxes() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("📮 Mailboxes"))
	b.WriteString("\n\n")

	for i, mb := range m.mailboxes {
		if i == m.selectedMailbox {
			b.WriteString(selectedStyle.Render(fmt.Sprintf("> %s", mb)))
		} else {
			b.WriteString(fmt.Sprintf("  %s", mb))
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (m Model) formatMessage(msg *db.InboxMessage) string {
	var b strings.Builder

	b.WriteString(headerStyle.Render("From: "))
	b.WriteString(msg.FromID)
	b.WriteString("\n")

	b.WriteString(headerStyle.Render("To: "))
	b.WriteString(strings.Join(msg.ToIDs, ", "))
	b.WriteString("\n")

	b.WriteString(headerStyle.Render("Subject: "))
	b.WriteString(msg.Subject)
	b.WriteString("\n")

	b.WriteString(headerStyle.Render("Priority: "))
	b.WriteString(msg.Priority)
	b.WriteString("\n")

	b.WriteString(headerStyle.Render("Time: "))
	b.WriteString(msg.CreatedAt.Format("2006-01-02 15:04:05"))
	b.WriteString("\n")

	b.WriteString(strings.Repeat("─", 50))
	b.WriteString("\n\n")

	b.WriteString(msg.Body)

	return b.String()
}

func (m *Model) updateInboxTable() {
	rows := make([]table.Row, len(m.messages))
	for i, msg := range m.messages {
		status := " "
		if msg.Status == "unread" {
			status = "•"
		}

		priority := msg.Priority
		if msg.Priority == "urgent" {
			priority = "🚨"
		} else if msg.Priority == "high" {
			priority = "!"
		}

		subject := msg.Subject
		if len(subject) > 28 {
			subject = subject[:25] + "..."
		}

		timeAgo := formatTimeAgo(msg.CreatedAt)

		rows[i] = table.Row{
			status,
			msg.ID[:8],
			msg.FromID,
			subject,
			priority,
			timeAgo,
		}
	}
	m.inboxTable.SetRows(rows)
}

// Messages
type inboxMsg struct {
	messages []db.InboxMessage
	err      error
}

type statusMsg string

type errMsg struct {
	err error
}

func (m Model) refreshInbox() tea.Cmd {
	return func() tea.Msg {
		messages, err := m.db.GetInbox(m.identity, true)
		return inboxMsg{messages: messages, err: err}
	}
}

func (m Model) sendMessage(to, subject, body string) tea.Cmd {
	return func() tea.Msg {
		recipients := strings.Split(to, ",")
		for i := range recipients {
			recipients[i] = strings.TrimSpace(recipients[i])
		}

		msg := &db.Message{
			ID:        generateID(),
			FromID:    m.identity,
			Subject:   subject,
			Body:      body,
			Priority:  "normal",
			MsgType:   "message",
			CreatedAt: timeNow(),
		}

		if err := m.db.SendMessage(msg, recipients); err != nil {
			return errMsg{err: err}
		}

		return statusMsg("Message sent!")
	}
}
