package cliview

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	pb "github.com/lize-y/brick/client/api/llm/v1"
	"google.golang.org/grpc"
)

const (
	gap = "\n\n"
)

type (
	errMsg error
)

type tokenMsg string
type doneMsg struct{}

type model struct {
	viewport      viewport.Model
	messages      []string
	textarea      textarea.Model
	senderStyle   lipgloss.Style
	botStyle      lipgloss.Style
	err           error
	client        pb.LLMServiceClient
	conn          *grpc.ClientConn
	stream        pb.LLMService_GenerateStreamClient
	currentAnswer strings.Builder
	generating    bool
}

func InitialModel(conn *grpc.ClientConn) tea.Model {
	ta := textarea.New()
	ta.Placeholder = "Send a message..."
	ta.Focus()

	ta.Prompt = "┃ "
	ta.CharLimit = 280

	ta.SetWidth(30)
	ta.SetHeight(3)

	// Remove cursor line styling
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()

	ta.KeyMap.InsertNewline.SetEnabled(false)

	vp := viewport.New(30, 5)
	vp.SetContent(`Welcome to the LLM Chat!
Type a message and press Enter to send.`)

	return model{
		textarea:    ta,
		messages:    []string{},
		viewport:    vp,
		senderStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("5")),
		botStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("2")),
		err:         nil,
		client:      pb.NewLLMServiceClient(conn),
		conn:        conn,
	}
}

func (m model) Init() tea.Cmd {
	return textarea.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
	)

	m.textarea, tiCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		m.textarea.SetWidth(msg.Width)
		m.viewport.Height = msg.Height - m.textarea.Height() - lipgloss.Height(gap)

		if len(m.messages) > 0 {
			m.viewport.SetContent(lipgloss.NewStyle().Width(m.viewport.Width).Render(strings.Join(m.messages, gap)))
			m.viewport.GotoBottom()
		}

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			if m.generating {
				return m, nil
			}
			v := m.textarea.Value()
			if v == "" {
				return m, nil
			}
			m.messages = append(m.messages, m.senderStyle.Render("You: ")+v)
			m.viewport.SetContent(lipgloss.NewStyle().Width(m.viewport.Width).Render(strings.Join(m.messages, gap)))
			m.textarea.Reset()
			m.viewport.GotoBottom()
			m.generating = true
			m.currentAnswer.Reset()

			// Start generation
			return m, startGeneration(m.client, v)
		}

	case tokenMsg:
		token := string(msg)
		m.currentAnswer.WriteString(token)

		// Update the last message (which is the bot's response being built)
		// If it's the first token, we need to append a new message
		if m.currentAnswer.Len() == len(token) {
			m.messages = append(m.messages, m.botStyle.Render("Bot: ")+m.currentAnswer.String())
		} else {
			m.messages[len(m.messages)-1] = m.botStyle.Render("Bot: ") + m.currentAnswer.String()
		}

		m.viewport.SetContent(lipgloss.NewStyle().Width(m.viewport.Width).Render(strings.Join(m.messages, gap)))
		m.viewport.GotoBottom()

		// Continue receiving
		return m, waitForToken(m.stream)

	case doneMsg:
		m.generating = false
		m.stream = nil
		return m, nil

	case errMsg:
		m.err = msg
		m.messages = append(m.messages, m.botStyle.Render("Error: ")+msg.Error())
		m.viewport.SetContent(lipgloss.NewStyle().Width(m.viewport.Width).Render(strings.Join(m.messages, gap)))
		m.viewport.GotoBottom()
		m.generating = false
		return m, nil

	// Special message to set the stream in the model, returned by startGeneration
	case streamMsg:
		m.stream = msg.stream
		return m, waitForToken(m.stream)
	}

	return m, tea.Batch(tiCmd, vpCmd)
}

func (m model) View() string {
	return fmt.Sprintf(
		"%s%s%s",
		m.viewport.View(),
		gap,
		m.textarea.View(),
	)
}

// Custom message to pass the stream back to the model
type streamMsg struct {
	stream pb.LLMService_GenerateStreamClient
}

func startGeneration(client pb.LLMServiceClient, prompt string) tea.Cmd {
	return func() tea.Msg {
		ctx, _ := context.WithTimeout(context.Background(), 60*time.Second)
		// Note: cancel is not called here because we need the stream to stay open.
		// In a real app, we might want to manage context cancellation better.

		stream, err := client.GenerateStream(ctx, &pb.GenerateRequest{
			Prompt:    prompt,
			MaxTokens: 100, // Default max tokens
		})
		if err != nil {
			return errMsg(err)
		}
		return streamMsg{stream: stream}
	}
}

func waitForToken(stream pb.LLMService_GenerateStreamClient) tea.Cmd {
	return func() tea.Msg {
		resp, err := stream.Recv()
		if err == io.EOF {
			return doneMsg{}
		}
		if err != nil {
			return errMsg(err)
		}
		return tokenMsg(resp.Token)
	}
}
