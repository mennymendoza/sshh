package tui

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
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

	m := newModel(channel, room)
	p := tea.NewProgram(m)

	go readLoop(channel, p)

	writeMessage(channel, protocol.ClientMessage{Type: protocol.MsgJoin, Room: room})

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("run tui: %w", err)
	}
	return nil
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
			p.Send(lineMsg(fmt.Sprintf("(unparseable line: %s)", scanner.Text())))
			continue
		}
		p.Send(formatIncoming(msg))
	}
	p.Send(lineMsg("(disconnected)"))
}

func formatIncoming(msg protocol.ServerMessage) lineMsg {
	switch msg.Type {
	case protocol.MsgMessage:
		return lineMsg(fmt.Sprintf("[%s] %s: %s", msg.Room, msg.Sender, msg.Body))
	case protocol.MsgRooms:
		return lineMsg(fmt.Sprintf("rooms: %s", strings.Join(msg.Rooms, ", ")))
	case protocol.MsgError:
		return lineMsg(fmt.Sprintf("error: %s", msg.Error))
	case protocol.MsgAck:
		return lineMsg("(ok)")
	default:
		return lineMsg(fmt.Sprintf("(unknown message type: %s)", msg.Type))
	}
}

type lineMsg string

type model struct {
	channel ssh.Channel
	room    string
	input   textinput.Model
	lines   []string
}

func newModel(channel ssh.Channel, room string) model {
	ti := textinput.New()
	ti.Placeholder = "type a message, /rooms, or /join <room>"
	ti.Focus()
	return model{channel: channel, room: room, input: ti}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case lineMsg:
		m.lines = append(m.lines, string(msg))
		if len(m.lines) > 200 {
			m.lines = m.lines[len(m.lines)-200:]
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			text := strings.TrimSpace(m.input.Value())
			m.input.SetValue("")
			if text == "" {
				return m, nil
			}
			switch {
			case text == "/rooms":
				writeMessage(m.channel, protocol.ClientMessage{Type: protocol.MsgListRooms})
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
	var b strings.Builder
	for _, line := range m.lines {
		b.WriteString(line)
		b.WriteString("\n")
	}
	fmt.Fprintf(&b, "-- room: %s --\n", m.room)
	b.WriteString(m.input.View())
	return b.String()
}
