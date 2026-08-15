package tui

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/crypto/ssh"

	"github.com/mennymendoza/sshh/internal/emoji"
	"github.com/mennymendoza/sshh/internal/protocol"
	"github.com/mennymendoza/sshh/internal/sshclient"
)

func Run(addr, user, room string) error {
	client, channel, err := sshclient.Dial(addr, user)
	if err != nil {
		return err
	}
	defer client.Close()

	m := newModel(channel, user, room)
	p := tea.NewProgram(m, tea.WithAltScreen())

	go readLoop(channel, p)

	writeMessage(channel, protocol.ClientMessage{Type: protocol.MsgJoin, Room: room})

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("run tui: %w", err)
	}
	return nil
}

func RunOnce(addr, user, room, message string) error {
	client, channel, err := sshclient.Dial(addr, user)
	if err != nil {
		return err
	}
	defer client.Close()
	defer channel.Close()

	message = emoji.Resolve(message)

	writeMessage(channel, protocol.ClientMessage{Type: protocol.MsgJoin, Room: room, Quiet: true})
	writeMessage(channel, protocol.ClientMessage{Type: protocol.MsgSend, Room: room, Body: message})

	scanner := bufio.NewScanner(channel)
	for scanner.Scan() {
		var msg protocol.ServerMessage
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			return fmt.Errorf("decode server message: %w", err)
		}
		switch msg.Type {
		case protocol.MsgError:
			return fmt.Errorf("server error: %s", msg.Error)
		case protocol.MsgMessage:
			if msg.Room == room && msg.Sender == user && msg.Body == message {
				return nil
			}
		}
	}
	return scanner.Err()
}

func RunStream(addr, user, room string) error {
	client, channel, err := sshclient.Dial(addr, user)
	if err != nil {
		return err
	}
	defer client.Close()
	defer channel.Close()

	writeMessage(channel, protocol.ClientMessage{Type: protocol.MsgJoin, Room: room})

	scanner := bufio.NewScanner(channel)
	for scanner.Scan() {
		var msg protocol.ServerMessage
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			continue
		}
		if msg.Type == protocol.MsgMessage {
			fmt.Printf("%s: %s\n", msg.Sender, msg.Body)
		}
	}
	return scanner.Err()
}

func writeMessage(channel ssh.Channel, msg protocol.ClientMessage) {
	payload, err := protocol.Encode(msg)
	if err != nil {
		return
	}
	channel.Write(payload)
}

func readLoop(channel ssh.Channel, p *tea.Program) {
	scanner := bufio.NewScanner(channel)
	for scanner.Scan() {
		var msg protocol.ServerMessage
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			p.Send(lineMsg(dimStyle.Render(fmt.Sprintf("(unparseable line: %s)", scanner.Text()))))
			continue
		}
		p.Send(formatIncoming(msg))
	}
	p.Send(lineMsg(dimStyle.Render("(disconnected)")))
}

func formatIncoming(msg protocol.ServerMessage) lineMsg {
	switch msg.Type {
	case protocol.MsgMessage:
		return lineMsg(fmt.Sprintf("%s: %s", userStyle(msg.Sender).Render(msg.Sender), msg.Body))
	case protocol.MsgRooms:
		return lineMsg(infoStyle.Render(fmt.Sprintf("rooms: %s", strings.Join(msg.Rooms, ", "))))
	case protocol.MsgUsers:
		return lineMsg(infoStyle.Render(fmt.Sprintf("users in %s: %s", msg.Room, strings.Join(msg.Users, ", "))))
	case protocol.MsgError:
		return lineMsg(errorStyle.Render(fmt.Sprintf("error: %s", msg.Error)))
	case protocol.MsgAck:
		return lineMsg(okStyle.Render("(ok)"))
	case protocol.MsgUserJoined:
		return lineMsg(joinStyle.Render(fmt.Sprintf("→ %s joined %s", msg.Sender, msg.Room)))
	case protocol.MsgUserLeft:
		return lineMsg(leaveStyle.Render(fmt.Sprintf("← %s left %s", msg.Sender, msg.Room)))
	default:
		return lineMsg(dimStyle.Render(fmt.Sprintf("(unknown message type: %s)", msg.Type)))
	}
}

type lineMsg string

const (
	defaultWidth        = 76
	defaultVisibleLines = 20
	minVisibleLines     = 3
	fixedChromeLines    = 8
	inputHeight         = 3
	inputPromptWidth    = 2
	pickerVisibleRows   = 5
	pickerMaxWidth      = 54
)

type model struct {
	channel       ssh.Channel
	user          string
	room          string
	input         textarea.Model
	lines         []string
	width         int
	height        int
	scrollOffset  int
	pickerActive  bool
	pickerFilter  textinput.Model
	pickerResults []emoji.Entry
	pickerIndex   int
}

func newModel(channel ssh.Channel, user, room string) model {
	ta := textarea.New()
	ta.ShowLineNumbers = false
	ta.Placeholder = "type a message, or /help for commands"
	ta.SetPromptFunc(inputPromptWidth, func(lineIdx int) string {
		if lineIdx == 0 {
			return "> "
		}
		return "  "
	})
	ta.FocusedStyle.Prompt = promptStyle
	ta.BlurredStyle.Prompt = promptStyle
	ta.FocusedStyle.Placeholder = placeholderStyle
	ta.BlurredStyle.Placeholder = placeholderStyle
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.BlurredStyle.CursorLine = lipgloss.NewStyle()
	ta.SetHeight(inputHeight)
	ta.Focus()

	filter := textinput.New()
	filter.Prompt = "> "
	filter.Placeholder = "type to filter…"
	filter.PromptStyle = promptStyle
	filter.PlaceholderStyle = placeholderStyle

	return model{channel: channel, user: user, room: room, input: ta, pickerFilter: filter, width: defaultWidth}
}

func filterEmoji(filter string) []emoji.Entry {
	if filter == "" {
		return emoji.All()
	}
	return emoji.Search(filter)
}

func (m model) Init() tea.Cmd {
	return textarea.Blink
}

func (m model) visibleLines() int {
	if m.height <= 0 {
		return defaultVisibleLines
	}
	n := m.height - fixedChromeLines
	if n < minVisibleLines {
		return minVisibleLines
	}
	return n
}

func (m model) maxScrollOffset() int {
	return max(0, len(m.lines)-m.visibleLines())
}

func (m model) pickerContentWidth() int {
	return min(pickerMaxWidth, max(20, m.width-6))
}

func (m *model) closePicker() {
	m.pickerActive = false
	m.pickerResults = nil
	m.pickerFilter.Blur()
	m.pickerFilter.SetValue("")
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width - 2
		if m.width < 32 {
			m.width = 32
		}
		m.input.SetWidth(m.width - 2)
		m.pickerFilter.Width = m.pickerContentWidth() - 2 - lipgloss.Width(m.pickerFilter.Prompt)
		m.height = msg.Height
		m.scrollOffset = min(m.scrollOffset, m.maxScrollOffset())
		return m, nil

	case lineMsg:
		m.lines = append(m.lines, string(msg))
		if len(m.lines) > 200 {
			m.lines = m.lines[len(m.lines)-200:]
		}
		m.scrollOffset = min(m.scrollOffset, m.maxScrollOffset())
		return m, nil

	case tea.KeyMsg:
		if m.pickerActive {
			switch msg.Type {
			case tea.KeyCtrlC:
				return m, tea.Quit
			case tea.KeyEsc:
				m.closePicker()
				return m, nil
			case tea.KeyUp:
				if m.pickerIndex > 0 {
					m.pickerIndex--
				}
				return m, nil
			case tea.KeyDown:
				if m.pickerIndex < len(m.pickerResults)-1 {
					m.pickerIndex++
				}
				return m, nil
			case tea.KeyEnter:
				if len(m.pickerResults) > 0 {
					m.input.InsertString(m.pickerResults[m.pickerIndex].Char)
				}
				m.closePicker()
				return m, nil
			default:
				var cmd tea.Cmd
				m.pickerFilter, cmd = m.pickerFilter.Update(msg)
				m.pickerResults = filterEmoji(m.pickerFilter.Value())
				m.pickerIndex = 0
				return m, cmd
			}
		}

		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyUp:
			m.scrollOffset = min(m.scrollOffset+1, m.maxScrollOffset())
			return m, nil
		case tea.KeyDown:
			m.scrollOffset = max(m.scrollOffset-1, 0)
			return m, nil
		case tea.KeyEnter:
			text := strings.TrimSpace(m.input.Value())
			m.input.SetValue("")
			if text == "" {
				return m, nil
			}
			switch {
			case text == "/rooms":
				writeMessage(m.channel, protocol.ClientMessage{Type: protocol.MsgListRooms})
			case text == "/users":
				writeMessage(m.channel, protocol.ClientMessage{Type: protocol.MsgListUsers, Room: m.room})
			case text == "/clear":
				m.lines = nil
			case text == "/help":
				m.lines = append(m.lines,
					infoStyle.Render("available commands:"),
					infoStyle.Render("  /rooms          list all rooms"),
					infoStyle.Render("  /users          list users in the current room"),
					infoStyle.Render("  /join <room>    switch to a different room"),
					infoStyle.Render("  /emoji          browse emojis to insert, then type to filter"),
					infoStyle.Render("  :shortcode:     resolved to an emoji when the message is sent, e.g. :fire:"),
					infoStyle.Render("  /clear          clear the message log"),
					infoStyle.Render("  /help           show this message"),
					infoStyle.Render("  (anything else) send a message to the current room"),
				)
			case strings.HasPrefix(text, "/join "):
				m.room = strings.TrimSpace(strings.TrimPrefix(text, "/join "))
				writeMessage(m.channel, protocol.ClientMessage{Type: protocol.MsgJoin, Room: m.room})
			case text == "/emoji":
				m.pickerActive = true
				m.pickerResults = emoji.All()
				m.pickerIndex = 0
				m.pickerFilter.SetValue("")
				m.pickerFilter.Focus()
			default:
				writeMessage(m.channel, protocol.ClientMessage{Type: protocol.MsgSend, Room: m.room, Body: emoji.Resolve(text)})
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m model) View() string {
	header := titleStyle.Render(fmt.Sprintf("sshh client — connected as %s", m.user))

	total := len(m.lines)
	visible := m.visibleLines()
	start := total - visible - m.scrollOffset
	if start < 0 {
		start = 0
	}
	end := start + visible
	if end > total {
		end = total
	}
	shown := m.lines[start:end]

	var log strings.Builder
	if start > 0 {
		log.WriteString(dimStyle.Render("↑ scroll back"))
		log.WriteString("\n")
	}
	for _, line := range shown {
		log.WriteString(line)
		log.WriteString("\n")
	}

	parts := []string{
		header,
		"",
		strings.TrimRight(log.String(), "\n"),
		"",
		badgeStyle.Render(m.room),
	}
	if m.pickerActive {
		parts = append(parts, m.pickerView())
	}
	parts = append(parts, m.input.View())

	body := lipgloss.JoinVertical(lipgloss.Left, parts...)

	return frameStyle.Width(m.width).Render(body)
}

func (m model) pickerView() string {
	total := len(m.pickerResults)
	visible := min(pickerVisibleRows, total)

	start := m.pickerIndex - visible/2
	start = max(0, min(start, total-visible))
	end := start + visible

	rows := make([]string, 0, visible+1)
	for i := start; i < end; i++ {
		e := m.pickerResults[i]
		row := "  " + fmt.Sprintf("%s  %s", e.Char, e.Description)
		if i == m.pickerIndex {
			row = pickerSelectedStyle.Render(row)
		}
		rows = append(rows, row)
	}
	if total == 0 {
		rows = append(rows, dimStyle.Render("no matches"))
	}

	footer := fmt.Sprintf("%d/%d · esc cancel", m.pickerIndex+1, total)
	if total == 0 {
		footer = "esc cancel"
	}
	rows = append(rows, dimStyle.Render(footer))
	rows = append(rows, m.pickerFilter.View())
	return pickerStyle.Width(m.pickerContentWidth()).Render(strings.Join(rows, "\n"))
}
