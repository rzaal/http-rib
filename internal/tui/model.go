// Package tui implements the Bubble Tea application for http-rib.
package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/rzaal/http-rib/internal/capture"
	"github.com/rzaal/http-rib/internal/collection"
	"github.com/rzaal/http-rib/internal/curl"
	"github.com/rzaal/http-rib/internal/render"
)

type responseMsg struct {
	result   curl.Result
	captured map[string]string
	skipped  []string
}

type mode int

const (
	modeEnvSelect mode = iota
	modeMain
)

type focus int

const (
	focusCollection focus = iota
	focusParam
)

type Model struct {
	coll *collection.Collection

	mode mode

	flat     []collection.FlatItem
	cursor   int
	envNames []string
	envIdx   int // active env used for sending
	envPick  int // cursor position while in the env picker

	// params holds :name values keyed by param name, persisted for the
	// life of the process (shared across requests that reuse a name).
	params     map[string]string
	paramNames []string // :params for the currently selected request
	focus      focus
	paramIdx   int

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

	model := Model{
		coll:     coll,
		mode:     m,
		flat:     collection.Flatten(coll.Tree),
		envNames: envNames,
		envIdx:   envIdx,
		envPick:  envIdx,
		params:   make(map[string]string),
		spinner:  sp,
		status:   "ready",
	}
	model.refreshParamNames()
	return model
}

// refreshParamNames recomputes paramNames for the currently selected
// request and clamps focus/paramIdx so they stay valid after the selection
// changes. Previously entered param values are kept in m.params.
func (m *Model) refreshParamNames() {
	req := m.selectedRequest()
	if req == nil {
		m.paramNames = nil
	} else {
		m.paramNames = render.ExtractParamNames(req.URL, req.Query)
	}
	if len(m.paramNames) == 0 {
		m.focus = focusCollection
		m.paramIdx = 0
		m.layoutViewport()
		return
	}
	if m.paramIdx >= len(m.paramNames) {
		m.paramIdx = 0
	}
	m.layoutViewport()
}

// paramsBoxHeight is the total vertical space the params box occupies,
// including its border and the gap below it, or 0 when the current request
// has no :params (in which case no box is rendered at all).
func (m *Model) paramsBoxHeight() int {
	if len(m.paramNames) == 0 {
		return 0
	}
	return len(m.paramNames) + 2 + 1 // rows + border(top+bottom) + gap below
}

// layoutViewport recomputes the output viewport's size from the current
// terminal size and the params box (which grows/shrinks with the selected
// request), keeping the sidebar/params-box/output layout consistent.
func (m *Model) layoutViewport() {
	if !m.ready {
		return
	}
	vpW := m.width - sidebarWidth(m.width) - 6
	vpH := m.height - 8 - m.paramsBoxHeight()
	if vpW < 10 {
		vpW = 10
	}
	if vpH < 3 {
		vpH = 3
	}
	m.viewport.Width = vpW
	m.viewport.Height = vpH
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
		if !m.ready {
			m.viewport = viewport.New(10, 3)
			m.ready = true
		}
		m.layoutViewport()
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

		if m.focus == focusParam {
			return m.updateParamFocus(msg)
		}

		switch msg.String() {
		case "q":
			return m, tea.Quit
		case "tab":
			if len(m.paramNames) > 0 {
				m.focus = focusParam
				m.paramIdx = 0
			}
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				m.refreshParamNames()
			}
		case "down", "j":
			if m.cursor < len(m.flat)-1 {
				m.cursor++
				m.refreshParamNames()
			}
		case "enter":
			if req := m.selectedRequest(); req != nil && !m.sending {
				m.sending = true
				m.status = fmt.Sprintf("sending %s %s...", req.Method, req.URL)
				return m, tea.Batch(m.spinner.Tick, sendCmd(req, m.currentEnv(), m.coll.Dir, m.currentEnvName(), m.params))
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
				args := curl.BuildArgs(req, m.currentEnv(), m.params)
				m.status = "curl: " + curl.CommandLine(args)
			}
		case "r":
			if reloaded, err := collection.Load(m.coll.Dir); err == nil {
				m.coll = reloaded
				m.flat = collection.Flatten(reloaded.Tree)
				if m.cursor >= len(m.flat) {
					m.cursor = len(m.flat) - 1
				}
				m.refreshParamNames()
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
		if len(msg.captured) > 0 {
			if reloaded, err := collection.LoadEnvs(m.coll.Dir); err == nil {
				m.coll.Envs = reloaded
			}
			names := make([]string, 0, len(msg.captured))
			for k := range msg.captured {
				names = append(names, k)
			}
			sort.Strings(names)
			m.status += "  |  captured " + strings.Join(names, ", ")
		}
		if len(msg.skipped) > 0 {
			m.status += "  |  skipped " + strings.Join(msg.skipped, ", ")
		}
		m.viewport.SetContent(strings.TrimSpace(r.Body))
		m.viewport.GotoTop()
		return m, nil
	}

	return m, nil
}

// updateParamFocus handles key input while a :param editor above the output
// window is focused. Tab cycles to the next param, wrapping back to the
// collection after the last one; shift+tab cycles backward.
func (m Model) updateParamFocus(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if len(m.paramNames) == 0 {
		m.focus = focusCollection
		return m, nil
	}
	name := m.paramNames[m.paramIdx]

	switch msg.Type {
	case tea.KeyTab:
		m.paramIdx++
		if m.paramIdx >= len(m.paramNames) {
			m.paramIdx = 0
			m.focus = focusCollection
		}
	case tea.KeyShiftTab:
		m.paramIdx--
		if m.paramIdx < 0 {
			m.paramIdx = len(m.paramNames) - 1
		}
	case tea.KeyEsc:
		m.focus = focusCollection
	case tea.KeyEnter:
		if req := m.selectedRequest(); req != nil && !m.sending {
			m.sending = true
			m.status = fmt.Sprintf("sending %s %s...", req.Method, req.URL)
			return m, tea.Batch(m.spinner.Tick, sendCmd(req, m.currentEnv(), m.coll.Dir, m.currentEnvName(), m.params))
		}
	case tea.KeyBackspace:
		if v := m.params[name]; len(v) > 0 {
			m.params[name] = v[:len(v)-1]
		}
	case tea.KeySpace:
		m.params[name] += " "
	case tea.KeyRunes:
		m.params[name] += string(msg.Runes)
	}
	return m, nil
}

func sendCmd(req *collection.Request, env collection.Env, collDir, envName string, params map[string]string) tea.Cmd {
	return func() tea.Msg {
		args := curl.BuildArgs(req, env, params)
		result := curl.Run(context.Background(), args)

		msg := responseMsg{result: result}
		if result.Err == nil && req.Post != nil && len(req.Post.Captures) > 0 && envName != "" {
			resp := capture.Parse(result.Body)
			values, skipped := capture.Extract(req.Post.Captures, resp)
			for k, v := range values {
				if err := collection.WriteEnvVar(collDir, envName, k, v); err != nil {
					skipped = append(skipped, k+" (write failed)")
					delete(values, k)
				}
			}
			msg.captured = values
			msg.skipped = skipped
		}
		return msg
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

	right := m.renderMain()
	if paramsBox := m.renderParamsBox(); paramsBox != "" {
		right = lipgloss.JoinVertical(lipgloss.Left, paramsBox, "", right)
	}

	top := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, right)
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

// renderParamsBox renders the :param table in its own bordered box, sized
// to match the output window below it. Each row is "name   value", with
// names padded to a shared column width. Returns "" when the current
// request has no :params, so the caller can skip the box entirely.
func (m Model) renderParamsBox() string {
	if len(m.paramNames) == 0 {
		return ""
	}

	nameWidth := 0
	for _, name := range m.paramNames {
		if len(name) > nameWidth {
			nameWidth = len(name)
		}
	}

	var rows []string
	for i, name := range m.paramNames {
		val := m.params[name]

		var row string
		if m.focus == focusParam && i == m.paramIdx {
			display := val
			if display == "" {
				display = "-"
			}
			row = paramRowFocusedStyle.Render(fmt.Sprintf("%-*s   %s", nameWidth, name, display))
		} else {
			display := dimStyle.Render("-")
			if val != "" {
				display = val
			}
			row = paramNameStyle.Render(fmt.Sprintf("%-*s", nameWidth, name)) + "   " + display
		}
		rows = append(rows, row)
	}

	content := strings.Join(rows, "\n")
	return paramsBoxStyle.Width(m.viewport.Width + 2).Render(content)
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
	if len(m.paramNames) > 0 {
		keys = "tab params · " + keys
	}
	if m.focus == focusParam {
		keys = "type to edit · tab/shift+tab cycle · esc/enter done · ctrl+c quit"
	}
	return m.status + "  |  " + keys
}
