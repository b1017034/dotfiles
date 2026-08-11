// Package ui は Bubble Tea による chezmoi の操作画面。
package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/b1017034/dotfiles/internal/chezmoi"
	"github.com/b1017034/dotfiles/internal/run"
)

type mode int

const (
	modeList mode = iota
	modeRunning
	modeConfirm
	modeHelp
)

type action int

const (
	actApply action = iota
	actDiff
	actReAdd
)

func (a action) String() string {
	switch a {
	case actApply:
		return "apply"
	case actDiff:
		return "diff"
	case actReAdd:
		return "re-add"
	}
	return "?"
}

type statusMsg struct {
	entries []chezmoi.Entry
	err     error
}
type logMsg run.Event
type doneMsg struct{}

type model struct {
	cz *chezmoi.Client

	entries []chezmoi.Entry
	loading bool
	errMsg  string

	cursor   int
	selected map[int]bool

	mode    mode
	prev    mode
	pending action
	// apply でターゲット側の変更を上書きしてよいか。確認を経て立つ。
	force bool

	spin    spinner.Model
	logs    []run.Event
	eventCh chan run.Event
	running bool
	title   string
	act     action
	// 実行結果の表示を末尾から何行さかのぼっているか
	logScroll int

	width  int
	height int
}

func Run(cz *chezmoi.Client) error {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(colAccent)

	m := model{
		cz:       cz,
		spin:     s,
		loading:  true,
		selected: map[int]bool{},
		width:    90,
		height:   30,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.spin.Tick, m.refreshCmd())
}

func (m model) refreshCmd() tea.Cmd {
	cz := m.cz
	return func() tea.Msg {
		entries, err := cz.Status()
		return statusMsg{entries: entries, err: err}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		return m, cmd

	case statusMsg:
		m.entries = msg.entries
		m.errMsg = ""
		if msg.err != nil {
			m.errMsg = msg.err.Error()
		}
		m.loading = false
		m.selected = map[int]bool{}
		if m.cursor >= len(m.entries) {
			m.cursor = max(0, len(m.entries)-1)
		}
		return m, nil

	case logMsg:
		m.logs = append(m.logs, run.Event(msg))
		return m, waitEvent(m.eventCh)

	case doneMsg:
		m.running = false
		m.loading = true
		return m, m.refreshCmd()

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if key == "ctrl+c" {
		return m, tea.Quit
	}

	switch m.mode {

	case modeHelp:
		m.mode = m.prev
		return m, nil

	case modeRunning:
		if m.running {
			return m, nil
		}
		switch key {
		case "up", "k":
			if m.logScroll < len(m.logs)-m.logWindow() {
				m.logScroll++
			}
		case "down", "j":
			if m.logScroll > 0 {
				m.logScroll--
			}
		default:
			m.mode = modeList
		}
		return m, nil

	case modeConfirm:
		switch key {
		case "y", "Y":
			return m.start(m.pending)
		default:
			m.mode = modeList
			return m, nil
		}
	}

	// modeList
	switch key {
	case "q", "esc":
		return m, tea.Quit

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.entries)-1 {
			m.cursor++
		}
	case "home", "g":
		m.cursor = 0
	case "end", "G":
		m.cursor = max(0, len(m.entries)-1)

	case " ", "x":
		if len(m.entries) == 0 {
			break
		}
		if m.selected[m.cursor] {
			delete(m.selected, m.cursor)
		} else {
			m.selected[m.cursor] = true
		}
	case "a":
		if len(m.selected) == len(m.entries) {
			m.selected = map[int]bool{}
		} else {
			for i := range m.entries {
				m.selected[i] = true
			}
		}

	case "A":
		// chezmoi 自身も上書き確認を出すが、この画面では答えられないので先に聞く
		if len(chezmoi.LocallyModified(m.entries, m.cfgPaths())) > 0 {
			m.pending = actApply
			m.force = true
			m.mode = modeConfirm
			return m, nil
		}
		m.force = false
		return m.start(actApply)
	case "d":
		return m.start(actDiff)
	case "R":
		m.pending = actReAdd
		m.mode = modeConfirm
		return m, nil

	case "r":
		m.loading = true
		return m, m.refreshCmd()

	case "?":
		m.prev = m.mode
		m.mode = modeHelp
		return m, nil
	}

	return m, nil
}

// 操作対象の index。選択が無ければカーソル行。
func (m model) targets() []int {
	var out []int
	for i := range m.entries {
		if m.selected[i] {
			out = append(out, i)
		}
	}
	if len(out) == 0 && len(m.entries) > 0 {
		out = []int{m.cursor}
	}
	return out
}

// 操作対象のパス。差分が無ければ nil (= 全体が対象)。
func (m model) cfgPaths() []string {
	var out []string
	for _, i := range m.targets() {
		out = append(out, m.entries[i].Path)
	}
	return out
}

func (m model) start(act action) (tea.Model, tea.Cmd) {
	paths := m.cfgPaths()

	ch := make(chan run.Event, 128)
	m.eventCh = ch
	m.mode = modeRunning
	m.running = true
	m.logs = nil
	m.title = act.String()
	m.act = act
	m.logScroll = 0

	cz, force := m.cz, m.force

	go func() {
		rep := run.Reporter(func(e run.Event) { ch <- e })
		switch act {
		case actApply:
			cz.Apply(rep, force, paths...) //nolint:errcheck // ログに出している
		case actDiff:
			cz.Diff(rep, paths...) //nolint:errcheck
		case actReAdd:
			cz.ReAdd(rep, paths...) //nolint:errcheck
		}
		close(ch)
	}()

	return m, tea.Batch(m.spin.Tick, waitEvent(ch))
}

func waitEvent(ch chan run.Event) tea.Cmd {
	if ch == nil {
		return nil
	}
	return func() tea.Msg {
		e, ok := <-ch
		if !ok {
			return doneMsg{}
		}
		return logMsg(e)
	}
}

func (m model) View() string {
	switch m.mode {
	case modeRunning:
		return m.viewRunning()
	case modeHelp:
		return m.viewHelp()
	default:
		return m.viewList()
	}
}

func (m model) viewList() string {
	var b strings.Builder

	b.WriteString(m.header())
	b.WriteString("\n\n")
	b.WriteString(m.table())
	b.WriteString("\n")
	b.WriteString(m.detail())

	if m.mode == modeConfirm {
		b.WriteString("\n")
		b.WriteString(m.confirmBox())
	}

	b.WriteString("\n")
	b.WriteString(m.footer())
	return b.String()
}

func (m model) header() string {
	left := titleStyle.Render("dotfiles")

	sel := ""
	if n := len(m.selected); n > 0 {
		sel = selectedStyle.Render(fmt.Sprintf(" · %d selected", n))
	}
	right := subtleStyle.Render("chezmoi") + sel

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right) - 1
	if gap < 1 {
		gap = 1
	}
	return " " + left + strings.Repeat(" ", gap) + right
}

func (m model) table() string {
	if m.errMsg != "" {
		return " " + errStyle.Render(m.errMsg) + "\n"
	}
	if m.loading {
		return " " + subtleStyle.Render("chezmoi status を確認中…") + "\n"
	}
	if len(m.entries) == 0 {
		return " " + okStyle.Render("✔ 差分はありません") + "  " +
			subtleStyle.Render("（この状態で A / d を押すと全体が対象になります）") + "\n"
	}

	pw := 4
	for _, e := range m.entries {
		if w := lipgloss.Width(chezmoi.Short(e.Path)); w > pw {
			pw = w
		}
	}

	var b strings.Builder
	b.WriteString(headerStyle.Render(fmt.Sprintf("   %s  %s %s",
		pad("PATH", pw), pad("CODE", 6), "STATE")))
	b.WriteString("\n")

	for i, e := range m.entries {
		b.WriteString(fmt.Sprintf("%s%s %s  %s %s\n",
			m.cursorCell(i), m.markCell(i), m.nameCell(i, chezmoi.Short(e.Path), pw),
			pad(codeStyle(e), 6), entryLabel(e)))
	}
	return b.String()
}

func (m model) cursorCell(i int) string {
	if i == m.cursor {
		return cursorStyle.Render("❯ ")
	}
	return "  "
}

func (m model) markCell(i int) string {
	if m.selected[i] {
		return selectedStyle.Render("◉")
	}
	return skipStyle.Render("○")
}

func (m model) nameCell(i int, s string, w int) string {
	if i == m.cursor {
		return cursorStyle.Render(pad(s, w))
	}
	return textStyle.Render(pad(s, w))
}

func codeStyle(e chezmoi.Entry) string {
	code := strings.ReplaceAll(e.Code(), " ", "·")
	if e.Dirty() {
		return warnStyle.Render(code)
	}
	return subtleStyle.Render(code)
}

func entryLabel(e chezmoi.Entry) string {
	switch e.Target {
	case 'A':
		return warnStyle.Render(e.Label())
	case 'M', 'R':
		return infoStyle.Render(e.Label())
	case 'D':
		return errStyle.Render(e.Label())
	}
	return subtleStyle.Render(e.Label())
}

func (m model) detail() string {
	if len(m.entries) == 0 || m.cursor >= len(m.entries) {
		return m.box("chezmoi", subtleStyle.Render(
			"A で apply、d で diff、R でターゲット側の変更をソースに取り込みます"))
	}
	e := m.entries[m.cursor]
	body := fmt.Sprintf("%s  %s\n%s  %s\n%s  %s",
		subtleStyle.Render(pad("path", 7)), e.Path,
		subtleStyle.Render(pad("code", 7)), e.Code(),
		subtleStyle.Render(pad("state", 7)), entryLabel(e))
	return m.box(chezmoi.Short(e.Path), body)
}

func (m model) box(title, body string) string {
	w := m.width - 4
	if w < 20 {
		w = 20
	}
	return boxStyle.Width(w).Render(
		lipgloss.NewStyle().Foreground(colAccent).Bold(true).Render(title) + "\n" + body)
}

func (m model) confirmBox() string {
	var names []string
	var headline, note string

	switch m.pending {
	case actReAdd:
		for _, p := range m.cfgPaths() {
			names = append(names, chezmoi.Short(p))
		}
		if len(names) == 0 {
			names = []string{"(管理対象すべて)"}
		}
		headline = "ソースに取り込みます"
		note = "ターゲット側の内容でリポジトリを上書きします   "
	case actApply:
		for _, p := range chezmoi.LocallyModified(m.entries, m.cfgPaths()) {
			names = append(names, chezmoi.Short(p))
		}
		headline = "ターゲット側の変更を破棄します"
		note = "以下は chezmoi の外で編集されています。残すなら R で取り込んでください   "
	}

	w := m.width - 4
	if w < 20 {
		w = 20
	}
	body := errStyle.Render(headline) + "\n" +
		textStyle.Render(strings.Join(names, ", ")) + "\n" +
		subtleStyle.Render(note) +
		keyStyle.Render("y") + subtleStyle.Render(" 実行   ") +
		keyStyle.Render("n") + subtleStyle.Render(" 中止")
	return confirmBoxStyle.Width(w).Render(body)
}

func (m model) footer() string {
	keys := [][2]string{
		{"↑↓", "移動"},
		{"space", "選択"},
		{"a", "全選択"},
		{"A", "apply"},
		{"d", "diff"},
		{"R", "re-add"},
		{"r", "更新"},
		{"?", "ヘルプ"},
		{"q", "終了"},
	}

	var parts []string
	for _, k := range keys {
		parts = append(parts, keyStyle.Render(k[0])+" "+descStyle.Render(k[1]))
	}
	return " " + strings.Join(parts, descStyle.Render("  ·  "))
}

func (m model) viewRunning() string {
	var b strings.Builder
	b.WriteString(m.header())
	b.WriteString("\n\n")

	if m.running {
		b.WriteString(" " + m.spin.View() + infoStyle.Render(" "+m.title+" 実行中…"))
	} else {
		b.WriteString(" " + okStyle.Render("✔ "+m.title+" 完了"))
	}
	b.WriteString("\n\n")

	maxLines := m.logWindow()
	logs := m.logs
	end := max(len(logs)-m.logScroll, min(maxLines, len(logs)))
	start := max(end-maxLines, 0)
	for _, e := range logs[start:end] {
		line := renderEvent(e)
		if m.act == actDiff && e.Level == run.LevelInfo {
			line = renderDiffLine(e.Text)
		}
		b.WriteString("  " + line + "\n")
	}

	b.WriteString("\n")
	switch {
	case m.running:
		b.WriteString(" " + descStyle.Render("完了までお待ちください"))
	case len(m.logs) > maxLines:
		pos := ""
		if m.logScroll > 0 {
			pos = descStyle.Render(fmt.Sprintf("  (末尾から %d 行上)", m.logScroll))
		}
		b.WriteString(" " + keyStyle.Render("↑↓") + descStyle.Render(" スクロール · ") +
			keyStyle.Render("他のキー") + descStyle.Render(" で一覧に戻る") + pos)
	default:
		b.WriteString(" " + keyStyle.Render("任意のキー") + descStyle.Render(" で一覧に戻る"))
	}
	return b.String()
}

func (m model) logWindow() int {
	n := m.height - 8
	if n < 5 {
		n = 5
	}
	return n
}

func renderEvent(e run.Event) string {
	switch e.Level {
	case run.LevelOK:
		return okStyle.Render(e.Text)
	case run.LevelWarn:
		return warnStyle.Render(e.Text)
	case run.LevelErr:
		return errStyle.Render(e.Text)
	case run.LevelSkip:
		return skipStyle.Render(e.Text)
	default:
		return textStyle.Render(e.Text)
	}
}

// unified diff の 1 行を色分けする。run.Stream が行頭に足す 2 スペースを除いて判定する。
func renderDiffLine(s string) string {
	line := strings.TrimPrefix(s, "  ")
	switch {
	case strings.HasPrefix(line, "diff --git"):
		return diffFileStyle.Render(s)
	case strings.HasPrefix(line, "+++"), strings.HasPrefix(line, "---"),
		strings.HasPrefix(line, "index "), strings.HasPrefix(line, "old mode"),
		strings.HasPrefix(line, "new mode"), strings.HasPrefix(line, "new file"),
		strings.HasPrefix(line, "deleted file"), strings.HasPrefix(line, "rename "),
		strings.HasPrefix(line, "similarity "):
		return subtleStyle.Render(s)
	case strings.HasPrefix(line, "@@"):
		return diffHunkStyle.Render(s)
	case strings.HasPrefix(line, "+"):
		return diffAddStyle.Render(s)
	case strings.HasPrefix(line, "-"):
		return diffDelStyle.Render(s)
	}
	return subtleStyle.Render(s)
}

func (m model) viewHelp() string {
	rows := [][2]string{
		{"↑ / k", "上へ"},
		{"↓ / j", "下へ"},
		{"g / G", "先頭 / 末尾"},
		{"space", "選択のトグル"},
		{"a", "全選択 / 全解除"},
		{"", ""},
		{"A", "apply      chezmoi apply"},
		{"d", "diff       chezmoi diff"},
		{"R", "re-add     ターゲットの変更をソースへ。確認あり"},
		{"", ""},
		{"r", "状態を再取得"},
		{"?", "このヘルプ"},
		{"q", "終了"},
	}

	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Foreground(colAccent).Bold(true).Render("キー操作"))
	b.WriteString("\n\n")
	for _, r := range rows {
		if r[0] == "" {
			b.WriteString("\n")
			continue
		}
		b.WriteString(fmt.Sprintf("  %s  %s\n",
			keyStyle.Render(pad(r[0], 10)), descStyle.Render(r[1])))
	}
	b.WriteString("\n")
	b.WriteString(subtleStyle.Render("選択が無いときはカーソル行が対象になります。"))
	b.WriteString("\n")
	b.WriteString(subtleStyle.Render("差分が無いときは、管理対象すべてが対象になります。"))
	b.WriteString("\n")
	b.WriteString(subtleStyle.Render("設定ファイルの追加は chezmoi add <path> を使ってください。"))

	w := m.width - 4
	if w < 20 {
		w = 20
	}
	return "\n" + boxStyle.Width(w).Render(b.String()) + "\n\n " +
		keyStyle.Render("任意のキー") + descStyle.Render(" で戻る")
}

func pad(s string, w int) string {
	diff := w - lipgloss.Width(s)
	if diff <= 0 {
		return s
	}
	return s + strings.Repeat(" ", diff)
}
