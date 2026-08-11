package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/b1017034/dotfiles/internal/chezmoi"
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

	if _, err := cz.SourceDir(); err != nil {
		return fmt.Errorf("%w\n  bootstrap.sh / bootstrap.ps1 を実行して chezmoi を設定してください", err)
	}

	if len(rest) == 0 {
		return ui.Run(cz)
	}

	cmd, cmdArgs := rest[0], rest[1:]
	rep := run.Reporter(func(e run.Event) { fmt.Println(colorize(e)) })

	switch cmd {
	case "status":
		return printStatus(cz)

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

func printStatus(cz *chezmoi.Client) error {
	entries, err := cz.Status()
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		fmt.Println("差分はありません")
		return nil
	}
	for _, e := range entries {
		fmt.Printf("%s  %s %s\n", e.Code(), pad(chezmoi.Short(e.Path), 40), e.Label())
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
	fmt.Print(`dotfiles - chezmoi のフロントエンド

  dotfiles                 管理画面 (TUI) を起動
  dotfiles status          chezmoi status
  dotfiles apply [path...] chezmoi apply (-y で確認省略)
  dotfiles diff  [path...] chezmoi diff

パッケージは chezmoi apply 時に .chezmoiscripts が導入する。
定義は home/.chezmoidata/packages.toml。
設定ファイルの追加は chezmoi add <path> を使ってください。
`)
}
