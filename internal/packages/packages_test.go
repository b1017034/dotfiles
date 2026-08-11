package packages

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// write は sourceDir に見立てた一時ディレクトリへ packages.toml を書く。
func write(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, filepath.FromSlash(Name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

const sample = `
[[packages]]
name = "both"
description = "両 OS 対応"
brew = "both-brew"
winget = "Both.Winget"

[[packages]]
name = "maconly"
brew = "mac-brew"

[[packages]]
name = "winonly"
winget = "Win.Only"

[[packages]]
name = "scripted"
brew = "scripted-brew"

  [packages.windows]
  script = "irm https://example.invalid/i.ps1 | iex"
  check = "scripted"
`

// TestLoadResolvesPerOS は、実行中の OS 向けに ID / Script が解決され、
// 他 OS 専用のものも Supported=false として全件返ることを確かめる。
func TestLoadResolvesPerOS(t *testing.T) {
	pkgs, err := Load(write(t, sample))
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) != 4 {
		t.Fatalf("全件返るはず: got %d", len(pkgs))
	}

	byName := map[string]Package{}
	for _, p := range pkgs {
		byName[p.Name] = p
	}

	// Hosts は OS に依らない
	if got := byName["both"].HostsLabel(); got != "macOS / Windows" {
		t.Errorf("both.HostsLabel = %q", got)
	}
	if got := byName["maconly"].HostsLabel(); got != "macOS" {
		t.Errorf("maconly.HostsLabel = %q", got)
	}
	// script しか持たない OS も Hosts に数える
	if got := byName["scripted"].HostsLabel(); got != "macOS / Windows" {
		t.Errorf("scripted.HostsLabel = %q", got)
	}

	type want struct {
		id        string
		script    string
		supported bool
	}
	var wants map[string]want
	switch runtime.GOOS {
	case "darwin":
		wants = map[string]want{
			"both":     {id: "both-brew", supported: true},
			"maconly":  {id: "mac-brew", supported: true},
			"winonly":  {supported: false},
			"scripted": {id: "scripted-brew", supported: true},
		}
	case "windows":
		wants = map[string]want{
			"both":     {id: "Both.Winget", supported: true},
			"maconly":  {supported: false},
			"winonly":  {id: "Win.Only", supported: true},
			"scripted": {script: "irm https://example.invalid/i.ps1 | iex", supported: true},
		}
	default:
		// brew も winget も無い OS では何も導入できない
		wants = map[string]want{
			"both": {supported: false}, "maconly": {supported: false},
			"winonly": {supported: false}, "scripted": {supported: false},
		}
	}

	for name, w := range wants {
		p := byName[name]
		if p.ID != w.id {
			t.Errorf("%s.ID = %q, want %q", name, p.ID, w.id)
		}
		if p.Script != w.script {
			t.Errorf("%s.Script = %q, want %q", name, p.Script, w.script)
		}
		if p.Supported() != w.supported {
			t.Errorf("%s.Supported = %v, want %v", name, p.Supported(), w.supported)
		}
		if w.script != "" && !p.ViaScript() {
			t.Errorf("%s は ViaScript であるはず", name)
		}
	}
}

// TestLoadRejectsBadInput は、静かに壊れるより読み込み時に落ちることを確かめる。
func TestLoadRejectsBadInput(t *testing.T) {
	cases := []struct {
		label string
		body  string
		want  string
	}{
		{
			"name なし",
			"[[packages]]\nbrew = \"x\"\n",
			"name がありません",
		},
		{
			"name 重複",
			"[[packages]]\nname = \"x\"\nbrew = \"a\"\n\n[[packages]]\nname = \"x\"\nbrew = \"b\"\n",
			"重複しています",
		},
		{
			// script だけあって check が無いと導入済み判定ができない
			"check なし",
			"[[packages]]\nname = \"x\"\n\n  [packages." + hostKey() + "]\n  script = \"true\"\n",
			"script と check の両方が必要です",
		},
	}

	for _, c := range cases {
		t.Run(c.label, func(t *testing.T) {
			if hostKey() == "" {
				t.Skip("brew / winget を持つ OS でのみ意味がある")
			}
			_, err := Load(write(t, c.body))
			if err == nil {
				t.Fatal("エラーになるはず")
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Errorf("err = %q, want contains %q", err, c.want)
			}
		})
	}
}

// hostKey は実行中の OS に対応するサブテーブル名を返す。
func hostKey() string {
	switch runtime.GOOS {
	case "darwin":
		return "darwin"
	case "windows":
		return "windows"
	}
	return ""
}
