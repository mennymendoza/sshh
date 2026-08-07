package tui

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/crypto/ssh"

	"github.com/mennymendoza/sshh/internal/protocol"
)

func Run(addr, user, room string) error {
	config := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.Password("test")},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return fmt.Errorf("dial %s: %w", addr, err)
	}
	defer client.Close()

	channel, requests, err := client.OpenChannel(protocol.ChannelType, nil)
	if err != nil {
		return fmt.Errorf("open chat channel: %w", err)
	}
	go ssh.DiscardRequests(requests)

	m := newModel(channel, user, room)
	p := tea.NewProgram(m)

	go readLoop(channel, p)

	writeMessage(channel, protocol.ClientMessage{Type: protocol.MsgJoin, Room: room})

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("run tui: %w", err)
	}
	return nil
}

func RunOnce(addr, user, room, message string) error {
	config := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.Password("test")},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return fmt.Errorf("dial %s: %w", addr, err)
	}
	defer client.Close()

	channel, requests, err := client.OpenChannel(protocol.ChannelType, nil)
	if err != nil {
		return fmt.Errorf("open chat channel: %w", err)
	}
	defer channel.Close()
	go ssh.DiscardRequests(requests)

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
	config := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.Password("test")},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return fmt.Errorf("dial %s: %w", addr, err)
	}
	defer client.Close()

	channel, requests, err := client.OpenChannel(protocol.ChannelType, nil)
	if err != nil {
		return fmt.Errorf("open chat channel: %w", err)
	}
	defer channel.Close()
	go ssh.DiscardRequests(requests)

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
	payload, err := json.Marshal(msg)
	if err != nil {
		return
	}
	payload = append(payload, '\n')
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
	fixedChromeLines    = 6
)

type model struct {
	channel      ssh.Channel
	user         string
	room         string
	input        textinput.Model
	lines        []string
	width        int
	height       int
	scrollOffset int
}

func newModel(channel ssh.Channel, user, room string) model {
	ti := textinput.New()
	ti.Prompt = "> "
	ti.PromptStyle = promptStyle
	ti.PlaceholderStyle = placeholderStyle
	ti.Placeholder = "type a message, or /help for commands"
	ti.Focus()
	return model{channel: channel, user: user, room: room, input: ti, width: defaultWidth}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
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

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width - 4
		if m.width > 96 {
			m.width = 96
		}
		if m.width < 32 {
			m.width = 32
		}
		m.input.Width = m.width - 4
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
					infoStyle.Render("  /clear          clear the message log"),
					infoStyle.Render("  /help           show this message"),
					infoStyle.Render("  (anything else) send a message to the current room"),
				)
			case strings.HasPrefix(text, "/join "):
				m.room = strings.TrimSpace(strings.TrimPrefix(text, "/join "))
				writeMessage(m.channel, protocol.ClientMessage{Type: protocol.MsgJoin, Room: m.room})
			default:
				writeMessage(m.channel, protocol.ClientMessage{Type: protocol.MsgSend, Room: m.room, Body: text})
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

	body := lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		strings.TrimRight(log.String(), "\n"),
		"",
		badgeStyle.Render(m.room),
		m.input.View(),
	)

	return frameStyle.Width(m.width).Render(body)
}
