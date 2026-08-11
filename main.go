// dotfiles は chezmoi とパッケージマネージャのフロントエンド。
//
//	dotfiles                        管理画面 (TUI) を起動
//	dotfiles status                 パッケージと設定の状態を表示
//	dotfiles install <id|#|all>...  パッケージを導入
//	dotfiles uninstall <id|#|all>.. パッケージを削除
//	dotfiles apply [path...]        chezmoi apply
//	dotfiles diff  [path...]        chezmoi diff
package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/b1017034/dotfiles/internal/chezmoi"
	"github.com/b1017034/dotfiles/internal/packages"
	"github.com/b1017034/dotfiles/internal/run"
	"github.com/b1017034/dotfiles/internal/ui"
)

func main() {
	if err := start(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error: "+err.Error())
		os.Exit(1)
	}
}

func start(args []string) error {
	// --yes / -y はどこにあっても拾う
	yes := false
	var rest []string
	for _, a := range args {
		if a == "-y" || a == "--yes" {
			yes = true
			continue
		}
		rest = append(rest, a)
	}

	if len(rest) > 0 {
		switch rest[0] {
		case "help", "-h", "--help":
			printUsage()
			return nil
		}
	}

	cz, err := chezmoi.New()
	if err != nil {
		if errors.Is(err, chezmoi.ErrNotFound) {
			return errors.New("chezmoi が見つかりません。先に導入してください:\n" +
				"  macOS   brew install chezmoi\n" +
				"  Windows winget install --id twpayne.chezmoi --exact")
		}
		return err
	}

	sourceDir, err := cz.SourceDir()
	if err != nil {
		return fmt.Errorf("%w\n  bootstrap.sh / bootstrap.ps1 を実行して chezmoi を設定してください", err)
	}

	pkgs, err := packages.Load(sourceDir)
	if err != nil {
		return err
	}
	pm := packages.NewManager()

	if len(rest) == 0 {
		return ui.Run(cz, pm, pkgs)
	}

	cmd, cmdArgs := rest[0], rest[1:]
	rep := run.Reporter(func(e run.Event) { fmt.Println(colorize(e)) })

	switch cmd {
	case "status":
		return printStatus(cz, pm, pkgs)

	case "install", "uninstall":
		targets, err := resolve(pkgs, cmdArgs)
		if err != nil {
			return err
		}
		if cmd == "uninstall" && !yes {
			var names []string
			for _, p := range targets {
				names = append(names, p.Name)
			}
			if !confirm(fmt.Sprintf("削除します: %s よろしいですか?", strings.Join(names, " "))) {
				fmt.Println("中止しました")
				return nil
			}
		}
		return doPackages(pm, cmd, targets, rep)

	case "apply":
		// chezmoi はターゲットが直接編集されていると上書き確認を出すが、
		// ここは TTY を渡していないので答えられない。先に自前で聞く。
		entries, err := cz.Status()
		if err != nil {
			return err
		}
		dirty := chezmoi.LocallyModified(entries, cmdArgs)
		if len(dirty) > 0 && !yes {
			fmt.Println("以下は chezmoi の外で編集されています。apply すると変更は失われます:")
			for _, p := range dirty {
				fmt.Println("  " + chezmoi.Short(p))
			}
			fmt.Println("残したい場合は先に chezmoi re-add でソースへ取り込んでください。")
			if !confirm("上書きしますか?") {
				fmt.Println("中止しました")
				return nil
			}
		}
		return cz.Apply(rep, len(dirty) > 0, cmdArgs...)

	case "diff":
		return cz.Diff(rep, cmdArgs...)
	}

	return fmt.Errorf("不明なコマンド: %s", cmd)
}

// resolve は "all" / 名前 / 番号 をパッケージに解決する。
// この OS で導入できないものは all では黙って除き、名前で指定されたらエラーにする。
func resolve(pkgs []packages.Package, args []string) ([]packages.Package, error) {
	if len(args) == 0 {
		return nil, errors.New("対象を指定してください (名前 / 番号 / all)")
	}
	if len(args) == 1 && args[0] == "all" {
		var out []packages.Package
		for _, p := range pkgs {
			if p.Supported() {
				out = append(out, p)
			}
		}
		return out, nil
	}

	var out []packages.Package
	for _, a := range args {
		var found *packages.Package
		for i := range pkgs {
			if pkgs[i].Name == a {
				found = &pkgs[i]
				break
			}
		}
		if found == nil {
			var n int
			if _, err := fmt.Sscanf(a, "%d", &n); err == nil && n >= 1 && n <= len(pkgs) {
				found = &pkgs[n-1]
			}
		}
		if found == nil {
			return nil, fmt.Errorf("不明なパッケージ: %s", a)
		}
		if !found.Supported() {
			return nil, fmt.Errorf("%s は %s では導入できません (%s のみ)",
				found.Name, packages.HostLabel(), found.HostsLabel())
		}
		out = append(out, *found)
	}
	return out, nil
}

func doPackages(pm packages.Manager, cmd string, targets []packages.Package, rep run.Reporter) error {
	var firstErr error
	for _, p := range targets {
		if !p.ViaScript() && !pm.Available() {
			return fmt.Errorf("%s が見つかりません", pm.Name())
		}

		switch cmd {
		case "install":
			if packages.Installed(pm, p) {
				rep.Skip("[%s] 導入済みです", p.Name)
				continue
			}
			if err := packages.Install(pm, p, rep); err != nil {
				rep.Err("[%s] インストールに失敗しました: %v", p.Name, err)
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
			rep.OK("[%s] installed", p.Name)

		case "uninstall":
			if p.ViaScript() {
				rep.Skip("[%s] script で導入したものなので削除できません", p.Name)
				continue
			}
			if !packages.Installed(pm, p) {
				rep.Skip("[%s] 導入されていません", p.Name)
				continue
			}
			if err := packages.Uninstall(pm, p, rep); err != nil {
				rep.Err("[%s] アンインストールに失敗しました: %v", p.Name, err)
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
			rep.OK("[%s] uninstalled", p.Name)
		}
		packages.Forget(pm, p)
	}
	return firstErr
}

func printStatus(cz *chezmoi.Client, pm packages.Manager, pkgs []packages.Package) error {
	pm.Refresh()

	idw := 7
	for _, p := range pkgs {
		if lipgloss.Width(p.Name) > idw {
			idw = lipgloss.Width(p.Name)
		}
	}

	fmt.Printf("PACKAGES (%s / %s)\n", packages.HostLabel(), pm.Name())
	if len(pkgs) == 0 {
		fmt.Println("  (なし)")
	}
	for i, p := range pkgs {
		state := "not installed"
		switch {
		case !p.Supported():
			state = "— " + p.HostsLabel() + " のみ"
		case packages.Installed(pm, p):
			state = "installed"
		}
		fmt.Printf("  %3d  %s  %s %s\n", i+1, pad(p.Name, idw), pad(state, 18), p.Desc)
	}

	entries, err := cz.Status()
	if err != nil {
		return err
	}

	fmt.Printf("\nCONFIGS (chezmoi)\n")
	if len(entries) == 0 {
		fmt.Println("  差分はありません")
		return nil
	}
	for _, e := range entries {
		fmt.Printf("  %s  %s %s\n", e.Code(), pad(chezmoi.Short(e.Path), 40), e.Label())
	}
	return nil
}

func pad(s string, w int) string {
	if d := w - lipgloss.Width(s); d > 0 {
		return s + strings.Repeat(" ", d)
	}
	return s
}

func confirm(prompt string) bool {
	fmt.Printf("%s [y/N]: ", prompt)
	sc := bufio.NewScanner(os.Stdin)
	if !sc.Scan() {
		return false
	}
	a := strings.TrimSpace(sc.Text())
	return a == "y" || a == "Y"
}

func colorize(e run.Event) string {
	const reset = "\033[0m"
	var code string
	switch e.Level {
	case run.LevelOK:
		code = "\033[32m"
	case run.LevelWarn:
		code = "\033[33m"
	case run.LevelErr:
		code = "\033[31m"
	case run.LevelSkip:
		code = "\033[90m"
	default:
		return e.Text
	}
	return code + e.Text + reset
}

func printUsage() {
	fmt.Print(`dotfiles - chezmoi とパッケージマネージャのフロントエンド

  dotfiles                        管理画面 (TUI) を起動
  dotfiles status                 パッケージと設定の状態を表示
  dotfiles install <id|#|all>...  パッケージを導入
  dotfiles uninstall <id|#|all>.. パッケージを削除 (-y で確認省略)
  dotfiles apply [path...]        chezmoi apply
  dotfiles diff  [path...]        chezmoi diff

パッケージの定義は home/.chezmoidata/packages.toml。
設定ファイルの追加は chezmoi add <path> を使ってください。
`)
}
