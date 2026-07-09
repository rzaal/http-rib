// Package tui implements the Bubble Tea application for http-rib.
package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/rzaal/http-rib/internal/collection"
	"github.com/rzaal/http-rib/internal/curl"
)

type responseMsg struct {
	result curl.Result
}

type mode int

const (
	modeEnvSelect mode = iota
	modeMain
)

type Model struct {
	coll *collection.Collection

	mode mode

	flat     []collection.FlatItem
	cursor   int
	envNames []string
	envIdx   int // active env used for sending
	envPick  int // cursor position while in the env picker

	viewport viewport.Model
	spinner  spinner.Model
	sending  bool

	lastCommand string
	lastResult  *curl.Result
	status      string

	width, height int
	ready         bool
}

func New(coll *collection.Collection) Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot

	envNames := coll.Envs.Names()
	envIdx := 0
	defName, _ := coll.Envs.Default()
	for i, n := range envNames {
		if n == defName {
			envIdx = i
			break
		}
	}

	m := modeEnvSelect
	if len(envNames) <= 1 {
		m = modeMain
	}

	return Model{
		coll:     coll,
		mode:     m,
		flat:     collection.Flatten(coll.Tree),
		envNames: envNames,
		envIdx:   envIdx,
		envPick:  envIdx,
		spinner:  sp,
		status:   "ready",
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) currentEnvName() string {
	if len(m.envNames) == 0 {
		return ""
	}
	return m.envNames[m.envIdx]
}

func (m Model) currentEnv() collection.Env {
	if len(m.envNames) == 0 {
		return collection.Env{}
	}
	return m.coll.Envs.Get(m.envNames[m.envIdx])
}

func (m *Model) selectedRequest() *collection.Request {
	if m.cursor < 0 || m.cursor >= len(m.flat) {
		return nil
	}
	return m.flat[m.cursor].Item.Request
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		vpW := m.width - sidebarWidth(m.width) - 6
		vpH := m.height - 8
		if vpW < 10 {
			vpW = 10
		}
		if vpH < 3 {
			vpH = 3
		}
		if !m.ready {
			m.viewport = viewport.New(vpW, vpH)
			m.ready = true
		} else {
			m.viewport.Width = vpW
			m.viewport.Height = vpH
		}
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		if m.mode == modeEnvSelect {
			switch msg.String() {
			case "q":
				return m, tea.Quit
			case "up", "k":
				if m.envPick > 0 {
					m.envPick--
				}
			case "down", "j":
				if m.envPick < len(m.envNames)-1 {
					m.envPick++
				}
			case "enter":
				m.envIdx = m.envPick
				m.mode = modeMain
				m.status = "env -> " + m.currentEnvName()
			}
			return m, nil
		}

		switch msg.String() {
		case "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.flat)-1 {
				m.cursor++
			}
		case "enter":
			if req := m.selectedRequest(); req != nil && !m.sending {
				m.sending = true
				m.status = fmt.Sprintf("sending %s %s...", req.Method, req.URL)
				return m, tea.Batch(m.spinner.Tick, sendCmd(req, m.currentEnv()))
			}
		case "e":
			if len(m.envNames) > 0 {
				m.envPick = m.envIdx
				m.mode = modeEnvSelect
			}
		case "c":
			if m.lastCommand != "" {
				m.status = "curl: " + m.lastCommand
			} else if req := m.selectedRequest(); req != nil {
				args := curl.BuildArgs(req, m.currentEnv())
				m.status = "curl: " + curl.CommandLine(args)
			}
		case "r":
			if reloaded, err := collection.Load(m.coll.Dir); err == nil {
				m.coll = reloaded
				m.flat = collection.Flatten(reloaded.Tree)
				if m.cursor >= len(m.flat) {
					m.cursor = len(m.flat) - 1
				}
				m.status = "reloaded collection"
			} else {
				m.status = "reload failed: " + err.Error()
			}
		}
		return m, nil

	case spinner.TickMsg:
		if m.sending {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case responseMsg:
		m.sending = false
		r := msg.result
		m.lastResult = &r
		m.lastCommand = r.Command
		if r.Err != nil {
			m.status = "error: " + r.Err.Error()
		} else {
			m.status = fmt.Sprintf("done: %d in %s (%s)", r.StatusCode, r.TimeTotal, r.Duration.Round(1))
		}
		m.viewport.SetContent(strings.TrimSpace(r.Body))
		m.viewport.GotoTop()
		return m, nil
	}

	return m, nil
}

func sendCmd(req *collection.Request, env collection.Env) tea.Cmd {
	return func() tea.Msg {
		args := curl.BuildArgs(req, env)
		result := curl.Run(context.Background(), args)
		return responseMsg{result: result}
	}
}

func sidebarWidth(totalWidth int) int {
	w := totalWidth / 3
	if w < 24 {
		w = 24
	}
	if w > 40 {
		w = 40
	}
	return w
}

func (m Model) View() string {
	if !m.ready {
		return "loading..."
	}

	if m.mode == modeEnvSelect {
		return m.renderEnvPicker()
	}

	sw := sidebarWidth(m.width)
	sidebar := m.renderSidebar(sw)
	main := m.renderMain()

	top := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, main)
	statusBar := statusBarStyle.Width(m.width).Render(m.renderStatus())

	return lipgloss.JoinVertical(lipgloss.Left, top, statusBar)
}

func (m Model) renderSidebar(width int) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(m.coll.Manifest.Name))
	b.WriteString("\n\n")
	for i, fi := range m.flat {
		indent := strings.Repeat("  ", fi.Depth)
		label := fi.Item.Label
		if fi.Item.Request != nil {
			label = methodStyle.Render(fi.Item.Request.Method) + " " + label
		} else {
			label = label + "/"
		}
		line := indent + label
		if i == m.cursor {
			line = sidebarSelectedStyle.Render(indent + label)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	h := m.height - 4
	if h < 1 {
		h = 1
	}
	return sidebarStyle.Width(width).Height(h).Render(b.String())
}

func (m Model) renderMain() string {
	var b strings.Builder

	req := m.selectedRequest()
	if req != nil {
		b.WriteString(methodStyle.Render(req.Method) + " " + req.URL)
	} else {
		b.WriteString(dimStyle.Render("select a request"))
	}
	b.WriteString("  ")
	b.WriteString(dimStyle.Render("env: " + m.currentEnvName()))
	b.WriteString("\n\n")

	if m.sending {
		b.WriteString(m.spinner.View() + " sending...")
	} else {
		b.WriteString(m.viewport.View())
	}

	return mainStyle.Width(m.viewport.Width + 2).Height(m.viewport.Height + 2).Render(b.String())
}

func (m Model) renderEnvPicker() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(m.coll.Manifest.Name))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("select an environment"))
	b.WriteString("\n\n")

	for i, name := range m.envNames {
		line := "  " + name
		if i == m.envPick {
			line = sidebarSelectedStyle.Render("> " + name)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("↑/↓ choose · enter confirm · q quit"))

	box := mainStyle.Width(40).Render(b.String())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

func (m Model) renderStatus() string {
	keys := "↑/↓ nav · enter send · e env · c curl · r reload · q quit"
	return m.status + "  |  " + keys
}
