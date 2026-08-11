// Package ui は Bubble Tea による管理画面。
//
// 設定ファイルの状態は chezmoi に、パッケージの状態はパッケージマネージャに問い合わせる。
// この画面自身は状態を持たず、両者のフロントエンドに徹する。
package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/b1017034/dotfiles/internal/chezmoi"
	"github.com/b1017034/dotfiles/internal/packages"
	"github.com/b1017034/dotfiles/internal/run"
)

type mode int

const (
	modeList mode = iota
	modeRunning
	modeConfirm
	modeHelp
)

type tab int

const (
	tabPackages tab = iota
	tabConfigs
	numTabs
)

func (t tab) String() string {
	if t == tabConfigs {
		return "Configs"
	}
	return "Packages"
}

type action int

const (
	actInstall action = iota
	actUninstall
	actApply
	actDiff
	actReAdd
)

func (a action) String() string {
	switch a {
	case actInstall:
		return "install"
	case actUninstall:
		return "uninstall"
	case actApply:
		return "apply"
	case actDiff:
		return "diff"
	case actReAdd:
		return "re-add"
	}
	return "?"
}

// ---------------------------------------------------------------- messages

type pkgStatusMsg []bool
type cfgStatusMsg struct {
	entries []chezmoi.Entry
	err     error
}
type logMsg run.Event
type doneMsg struct{}

// ---------------------------------------------------------------- model

type model struct {
	cz   *chezmoi.Client
	pm   packages.Manager
	pkgs []packages.Package

	installed  []bool
	entries    []chezmoi.Entry
	pkgLoading bool
	cfgLoading bool
	cfgErr     string

	tab      tab
	cursor   [numTabs]int
	selected [numTabs]map[int]bool

	mode    mode
	prev    mode
	pending action
	// force は apply でターゲット側の変更を上書きしてよいか。
	// ターゲットが直接編集されている場合だけ、確認を経て立つ。
	force bool

	spin    spinner.Model
	logs    []run.Event
	eventCh chan run.Event
	running bool
	title   string

	width  int
	height int
}

// Run は管理画面を起動する。
func Run(cz *chezmoi.Client, pm packages.Manager, pkgs []packages.Package) error {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(colAccent)

	m := model{
		cz:         cz,
		pm:         pm,
		pkgs:       pkgs,
		spin:       s,
		pkgLoading: true,
		cfgLoading: true,
		width:      90,
		height:     30,
	}
	for i := range m.selected {
		m.selected[i] = map[int]bool{}
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.spin.Tick, m.refreshPkgCmd(), m.refreshCfgCmd())
}

func (m model) refreshPkgCmd() tea.Cmd {
	pm, pkgs := m.pm, m.pkgs
	return func() tea.Msg {
		pm.Refresh()
		out := make([]bool, len(pkgs))
		for i, p := range pkgs {
			out[i] = packages.Installed(pm, p)
		}
		return pkgStatusMsg(out)
	}
}

func (m model) refreshCfgCmd() tea.Cmd {
	cz := m.cz
	return func() tea.Msg {
		entries, err := cz.Status()
		return cfgStatusMsg{entries: entries, err: err}
	}
}

// ---------------------------------------------------------------- update

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

	case pkgStatusMsg:
		m.installed = []bool(msg)
		m.pkgLoading = false
		return m, nil

	case cfgStatusMsg:
		m.entries = msg.entries
		m.cfgErr = ""
		if msg.err != nil {
			m.cfgErr = msg.err.Error()
		}
		m.cfgLoading = false
		m.selected[tabConfigs] = map[int]bool{}
		if m.cursor[tabConfigs] >= len(m.entries) {
			m.cursor[tabConfigs] = max(0, len(m.entries)-1)
		}
		return m, nil

	case logMsg:
		m.logs = append(m.logs, run.Event(msg))
		return m, waitEvent(m.eventCh)

	case doneMsg:
		m.running = false
		m.pkgLoading = true
		m.cfgLoading = true
		return m, tea.Batch(m.refreshPkgCmd(), m.refreshCfgCmd())

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
			return m, nil // 実行中は入力を受け付けない
		}
		m.mode = modeList
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

	case "tab", "right", "l":
		m.tab = (m.tab + 1) % numTabs
	case "shift+tab", "left", "h":
		m.tab = (m.tab + numTabs - 1) % numTabs
	case "1":
		m.tab = tabPackages
	case "2":
		m.tab = tabConfigs

	case "up", "k":
		if m.cursor[m.tab] > 0 {
			m.cursor[m.tab]--
		}
	case "down", "j":
		if m.cursor[m.tab] < m.rows()-1 {
			m.cursor[m.tab]++
		}
	case "home", "g":
		m.cursor[m.tab] = 0
	case "end", "G":
		m.cursor[m.tab] = max(0, m.rows()-1)

	case " ", "x":
		i := m.cursor[m.tab]
		if m.rows() == 0 || !m.selectable(i) {
			break
		}
		if m.selected[m.tab][i] {
			delete(m.selected[m.tab], i)
		} else {
			m.selected[m.tab][i] = true
		}
	case "a":
		all := map[int]bool{}
		for i := 0; i < m.rows(); i++ {
			if m.selectable(i) {
				all[i] = true
			}
		}
		if len(m.selected[m.tab]) == len(all) {
			m.selected[m.tab] = map[int]bool{}
		} else {
			m.selected[m.tab] = all
		}

	case "i":
		if m.tab == tabPackages {
			return m.start(actInstall)
		}
	case "X":
		if m.tab == tabPackages {
			m.pending = actUninstall
			m.mode = modeConfirm
			return m, nil
		}
	case "A":
		if m.tab == tabConfigs {
			// ターゲットを直接いじった相手を黙って上書きしない。
			// chezmoi 自身も確認を出すが、この画面では答えられないので先に聞く。
			if len(chezmoi.LocallyModified(m.entries, m.cfgPaths())) > 0 {
				m.pending = actApply
				m.force = true
				m.mode = modeConfirm
				return m, nil
			}
			m.force = false
			return m.start(actApply)
		}
	case "d":
		if m.tab == tabConfigs {
			return m.start(actDiff)
		}
	case "R":
		if m.tab == tabConfigs {
			m.pending = actReAdd
			m.mode = modeConfirm
			return m, nil
		}

	case "r":
		m.pkgLoading = true
		m.cfgLoading = true
		return m, tea.Batch(m.refreshPkgCmd(), m.refreshCfgCmd())

	case "?":
		m.prev = m.mode
		m.mode = modeHelp
		return m, nil
	}

	return m, nil
}

// rows は現在のタブの行数。
func (m model) rows() int {
	if m.tab == tabConfigs {
		return len(m.entries)
	}
	return len(m.pkgs)
}

// selectable はその行を操作対象にできるかを返す。
// Packages タブでは、この OS で導入できないものは選べない。
func (m model) selectable(i int) bool {
	if m.tab != tabPackages {
		return true
	}
	return i < len(m.pkgs) && m.pkgs[i].Supported()
}

// targets は操作対象の index を返す。選択が無ければカーソル行。
func (m model) targets() []int {
	var out []int
	for i := 0; i < m.rows(); i++ {
		if m.selected[m.tab][i] && m.selectable(i) {
			out = append(out, i)
		}
	}
	if len(out) == 0 && m.rows() > 0 && m.selectable(m.cursor[m.tab]) {
		out = []int{m.cursor[m.tab]}
	}
	return out
}

// cfgPaths は Configs タブの操作対象パスを返す。差分が無ければ nil (= 全体が対象)。
func (m model) cfgPaths() []string {
	var out []string
	for _, i := range m.targets() {
		out = append(out, m.entries[i].Path)
	}
	return out
}

func (m model) start(act action) (tea.Model, tea.Cmd) {
	var (
		pkgTargets []packages.Package
		paths      []string
	)

	switch act {
	case actInstall, actUninstall:
		for _, i := range m.targets() {
			pkgTargets = append(pkgTargets, m.pkgs[i])
		}
		if len(pkgTargets) == 0 {
			return m, nil
		}
	default:
		paths = m.cfgPaths()
	}

	ch := make(chan run.Event, 128)
	m.eventCh = ch
	m.mode = modeRunning
	m.running = true
	m.logs = nil
	m.title = act.String()

	cz, pm, force := m.cz, m.pm, m.force

	go func() {
		rep := run.Reporter(func(e run.Event) { ch <- e })
		switch act {
		case actInstall:
			installPkgs(pm, pkgTargets, rep)
		case actUninstall:
			uninstallPkgs(pm, pkgTargets, rep)
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

func installPkgs(pm packages.Manager, targets []packages.Package, rep run.Reporter) {
	for _, p := range targets {
		if !p.ViaScript() && !pm.Available() {
			rep.Err("[%s] %s が見つかりません", p.Name, pm.Name())
			continue
		}
		if packages.Installed(pm, p) {
			rep.Skip("[%s] 導入済みです", p.Name)
			continue
		}
		if err := packages.Install(pm, p, rep); err != nil {
			rep.Err("[%s] インストールに失敗しました: %v", p.Name, err)
			continue
		}
		packages.Forget(pm, p)
		rep.OK("[%s] installed", p.Name)
	}
}

func uninstallPkgs(pm packages.Manager, targets []packages.Package, rep run.Reporter) {
	for _, p := range targets {
		if p.ViaScript() {
			rep.Skip("[%s] script で導入したものなので削除できません", p.Name)
			continue
		}
		if !pm.Available() {
			rep.Err("[%s] %s が見つかりません", p.Name, pm.Name())
			continue
		}
		if !packages.Installed(pm, p) {
			rep.Skip("[%s] 導入されていません", p.Name)
			continue
		}
		if err := packages.Uninstall(pm, p, rep); err != nil {
			rep.Err("[%s] アンインストールに失敗しました: %v", p.Name, err)
			continue
		}
		packages.Forget(pm, p)
		rep.OK("[%s] uninstalled", p.Name)
	}
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

// ---------------------------------------------------------------- view

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
	b.WriteString("\n")
	b.WriteString(m.tabs())
	b.WriteString("\n\n")

	if m.tab == tabConfigs {
		b.WriteString(m.tableConfigs())
	} else {
		b.WriteString(m.tablePackages())
	}
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
	if n := len(m.selected[m.tab]); n > 0 {
		sel = selectedStyle.Render(fmt.Sprintf(" · %d selected", n))
	}

	right := subtleStyle.Render(fmt.Sprintf("%s · %s · chezmoi",
		packages.HostLabel(), m.pm.Name())) + sel

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right) - 1
	if gap < 1 {
		gap = 1
	}
	return " " + left + strings.Repeat(" ", gap) + right
}

func (m model) tabs() string {
	// この OS で導入できるものが全体より少なければ "6/8" のように出す
	sup := 0
	for _, p := range m.pkgs {
		if p.Supported() {
			sup++
		}
	}
	pkgCount := fmt.Sprintf("%d", len(m.pkgs))
	if sup != len(m.pkgs) {
		pkgCount = fmt.Sprintf("%d/%d", sup, len(m.pkgs))
	}

	counts := []string{
		pkgCount,
		fmt.Sprintf("%d", len(m.entries)),
	}
	if m.cfgLoading {
		counts[1] = "…"
	}
	if m.pkgLoading {
		counts[0] = "…"
	}

	var parts []string
	for t := tabPackages; t < numTabs; t++ {
		label := fmt.Sprintf(" %d %s (%s) ", t+1, t, counts[t])
		if t == m.tab {
			parts = append(parts, tabActiveStyle.Render(label))
		} else {
			parts = append(parts, tabStyle.Render(label))
		}
	}
	return " " + strings.Join(parts, " ")
}

func (m model) tablePackages() string {
	if len(m.pkgs) == 0 {
		return " " + subtleStyle.Render("管理対象のパッケージがありません") + "\n"
	}

	idw := 4
	for _, p := range m.pkgs {
		if lipgloss.Width(p.Name) > idw {
			idw = lipgloss.Width(p.Name)
		}
	}

	var b strings.Builder
	b.WriteString(headerStyle.Render(fmt.Sprintf("   %s  %s %s",
		pad("PACKAGE", idw), pad("STATE", 18), "DESCRIPTION")))
	b.WriteString("\n")

	for i, p := range m.pkgs {
		// この OS で導入できないものは、存在だけ見えるようにグレーで出す
		if !p.Supported() {
			b.WriteString(fmt.Sprintf("%s%s %s  %s %s\n",
				m.cursorCell(i), skipStyle.Render("·"), skipStyle.Render(pad(p.Name, idw)),
				pad(skipStyle.Render("— "+p.HostsLabel()+" のみ"), 18), skipStyle.Render(p.Desc)))
			continue
		}

		state := skipStyle.Render("…")
		if !m.pkgLoading && i < len(m.installed) {
			if m.installed[i] {
				state = okStyle.Render("✔ installed")
			} else {
				state = warnStyle.Render("✘ not installed")
			}
		}
		b.WriteString(fmt.Sprintf("%s%s %s  %s %s\n",
			m.cursorCell(i), m.markCell(i), m.nameCell(i, p.Name, idw),
			pad(state, 18), subtleStyle.Render(p.Desc)))
	}
	return b.String()
}

func (m model) tableConfigs() string {
	if m.cfgErr != "" {
		return " " + errStyle.Render(m.cfgErr) + "\n"
	}
	if m.cfgLoading {
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
	if i == m.cursor[m.tab] {
		return cursorStyle.Render("❯ ")
	}
	return "  "
}

func (m model) markCell(i int) string {
	if m.selected[m.tab][i] {
		return selectedStyle.Render("◉")
	}
	return skipStyle.Render("○")
}

func (m model) nameCell(i int, s string, w int) string {
	if i == m.cursor[m.tab] {
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
	if m.tab == tabConfigs {
		if len(m.entries) == 0 || m.cursor[tabConfigs] >= len(m.entries) {
			return m.box("chezmoi", subtleStyle.Render(
				"A で apply、d で diff、R でターゲット側の変更をソースに取り込みます"))
		}
		e := m.entries[m.cursor[tabConfigs]]
		body := fmt.Sprintf("%s  %s\n%s  %s\n%s  %s",
			subtleStyle.Render(pad("path", 7)), e.Path,
			subtleStyle.Render(pad("code", 7)), e.Code(),
			subtleStyle.Render(pad("state", 7)), entryLabel(e))
		return m.box(chezmoi.Short(e.Path), body)
	}

	if len(m.pkgs) == 0 {
		return ""
	}
	p := m.pkgs[m.cursor[tabPackages]]

	var b strings.Builder
	if p.Desc != "" {
		b.WriteString(subtleStyle.Render(p.Desc))
		b.WriteString("\n")
	}
	b.WriteString(subtleStyle.Render(pad("対応", 7)) + "  " + p.HostsLabel() + "\n")

	switch {
	case !p.Supported():
		b.WriteString(subtleStyle.Render(pad("", 7)) + "  " +
			skipStyle.Render(packages.HostLabel()+" では導入できないため操作できません"))

	default:
		state := skipStyle.Render("確認中…")
		if !m.pkgLoading && m.cursor[tabPackages] < len(m.installed) {
			if m.installed[m.cursor[tabPackages]] {
				state = okStyle.Render("installed")
			} else {
				state = warnStyle.Render("not installed")
			}
		}
		if p.ViaScript() {
			// パッケージマネージャの管理外。削除できないことを明示する
			b.WriteString(fmt.Sprintf("%s  %s  %s\n",
				subtleStyle.Render(pad("script", 7)), p.Script, state))
			b.WriteString(subtleStyle.Render(pad("", 7)) + "  " +
				skipStyle.Render("script 導入のため uninstall は非対応"))
		} else {
			b.WriteString(fmt.Sprintf("%s  %s  %s",
				subtleStyle.Render(pad(m.pm.Name(), 7)), p.ID, state))
		}
	}

	return m.box(p.Name, b.String())
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
	case actUninstall:
		for _, i := range m.targets() {
			names = append(names, m.pkgs[i].Name)
		}
		headline = "アンインストールします"
		note = "パッケージを削除します。設定ファイルは chezmoi 管理のまま残ります   "
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
		{"tab", "タブ"},
		{"↑↓", "移動"},
		{"space", "選択"},
		{"a", "全選択"},
	}
	if m.tab == tabConfigs {
		keys = append(keys,
			[2]string{"A", "apply"},
			[2]string{"d", "diff"},
			[2]string{"R", "re-add"})
	} else {
		keys = append(keys,
			[2]string{"i", "install"},
			[2]string{"X", "uninstall"})
	}
	keys = append(keys,
		[2]string{"r", "更新"},
		[2]string{"?", "ヘルプ"},
		[2]string{"q", "終了"})

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

	maxLines := m.height - 8
	if maxLines < 5 {
		maxLines = 5
	}
	logs := m.logs
	if len(logs) > maxLines {
		logs = logs[len(logs)-maxLines:]
	}
	for _, e := range logs {
		b.WriteString("  " + renderEvent(e) + "\n")
	}

	b.WriteString("\n")
	if m.running {
		b.WriteString(" " + descStyle.Render("完了までお待ちください"))
	} else {
		b.WriteString(" " + keyStyle.Render("任意のキー") + descStyle.Render(" で一覧に戻る"))
	}
	return b.String()
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

func (m model) viewHelp() string {
	rows := [][2]string{
		{"tab / 1 2", "タブ切替 (Packages / Configs)"},
		{"↑ / k", "上へ"},
		{"↓ / j", "下へ"},
		{"g / G", "先頭 / 末尾"},
		{"space", "選択のトグル"},
		{"a", "全選択 / 全解除"},
		{"", ""},
		{"i", "install    パッケージを導入 (Packages)"},
		{"X", "uninstall  パッケージを削除。確認あり (Packages)"},
		{"", ""},
		{"A", "apply      chezmoi apply (Configs)"},
		{"d", "diff       chezmoi diff (Configs)"},
		{"R", "re-add     ターゲットの変更をソースへ。確認あり (Configs)"},
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
	b.WriteString(subtleStyle.Render("グレーの行は " + packages.HostLabel() +
		" で導入できないパッケージで、選択できません。"))
	b.WriteString("\n")
	b.WriteString(subtleStyle.Render("Configs で差分が無いときは、管理対象すべてが対象になります。"))
	b.WriteString("\n")
	b.WriteString(subtleStyle.Render("パッケージの追加は home/.chezmoidata/packages.toml を編集してください。"))
	b.WriteString("\n")
	b.WriteString(subtleStyle.Render("設定ファイルの追加は chezmoi add <path> を使ってください。"))

	w := m.width - 4
	if w < 20 {
		w = 20
	}
	return "\n" + boxStyle.Width(w).Render(b.String()) + "\n\n " +
		keyStyle.Render("任意のキー") + descStyle.Render(" で戻る")
}

// pad は表示幅を考慮して右埋めする。
func pad(s string, w int) string {
	diff := w - lipgloss.Width(s)
	if diff <= 0 {
		return s
	}
	return s + strings.Repeat(" ", diff)
}
