// Package packages は .chezmoidata/packages.toml の読み込みと、
// パッケージマネージャ (brew / winget) の操作を担当する。
//
// packages.toml は chezmoi のテンプレートデータでもあり、
// .chezmoiscripts の導入スクリプトと本パッケージが同じ定義を読む。
package packages

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/BurntSushi/toml"

	"github.com/b1017034/dotfiles/internal/run"
)

// Name は packages.toml の相対パス (chezmoi の source dir から見た位置)。
const Name = ".chezmoidata/packages.toml"

// Package は管理対象のパッケージ 1 件。
// 定義は OS 非依存で、実行中の OS 向けの情報が ID / Script に解決される。
type Package struct {
	// Name は表示名。OS に依らない。
	Name string
	// Desc は一覧に出す説明。
	Desc string

	// ID は実行中の OS のパッケージマネージャに渡す識別子。
	// brew なら formula 名、winget なら package ID。
	// この OS のパッケージマネージャで扱えないなら空。
	ID string

	// Script はパッケージマネージャに無いものを導入するコマンド。
	// Check は導入済み判定に使うコマンド名。ID が空のときだけ使う。
	Script string
	Check  string

	// Hosts は導入できる OS のラベル一覧 (表示用)。例: ["macOS", "Windows"]。
	Hosts []string
}

// Supported はこの OS で導入できるかを返す。
func (p Package) Supported() bool { return p.ID != "" || p.Script != "" }

// ViaScript はパッケージマネージャではなく script で導入するかを返す。
// この経路で入れたものはパッケージマネージャの管理外なので削除できない。
func (p Package) ViaScript() bool { return p.ID == "" && p.Script != "" }

// HostsLabel は導入できる OS を "macOS / Windows" のように返す。
func (p Package) HostsLabel() string {
	switch len(p.Hosts) {
	case 0:
		return "(なし)"
	case 1:
		return p.Hosts[0]
	default:
		return p.Hosts[0] + " / " + p.Hosts[1]
	}
}

// osEntry はパッケージマネージャに無いものの逃げ道。
type osEntry struct {
	Script string `toml:"script"`
	Check  string `toml:"check"`
}

type entry struct {
	Name    string   `toml:"name"`
	Desc    string   `toml:"description"`
	Brew    string   `toml:"brew"`
	Winget  string   `toml:"winget"`
	Darwin  *osEntry `toml:"darwin"`
	Windows *osEntry `toml:"windows"`
}

type file struct {
	Packages []entry `toml:"packages"`
}

// Load は source dir 配下の packages.toml を読む。
//
// 他 OS 専用のものも含めて全件返す。Supported が false のものは
// 一覧には出すが操作させない、という扱いを呼び出し側で行う。
func Load(sourceDir string) ([]Package, error) {
	path := filepath.Join(sourceDir, filepath.FromSlash(Name))

	var f file
	if _, err := toml.DecodeFile(path, &f); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%s がありません", path)
		}
		return nil, fmt.Errorf("%s の読み込みに失敗しました: %w", path, err)
	}

	out := make([]Package, 0, len(f.Packages))
	seen := map[string]bool{}
	for i, e := range f.Packages {
		if e.Name == "" {
			return nil, fmt.Errorf("%s の %d 番目に name がありません", path, i+1)
		}
		if seen[e.Name] {
			return nil, fmt.Errorf("%s でパッケージ名が重複しています: %s", path, e.Name)
		}
		seen[e.Name] = true

		p := Package{Name: e.Name, Desc: e.Desc}

		if e.Brew != "" || e.Darwin != nil {
			p.Hosts = append(p.Hosts, "macOS")
		}
		if e.Winget != "" || e.Windows != nil {
			p.Hosts = append(p.Hosts, "Windows")
		}

		var script *osEntry
		switch runtime.GOOS {
		case "darwin":
			p.ID, script = e.Brew, e.Darwin
		case "windows":
			p.ID, script = e.Winget, e.Windows
		}
		if p.ID == "" && script != nil {
			if script.Script == "" || script.Check == "" {
				return nil, fmt.Errorf("%s の %s は script と check の両方が必要です", path, e.Name)
			}
			p.Script, p.Check = script.Script, script.Check
		}

		out = append(out, p)
	}
	return out, nil
}

// ---------------------------------------------------------------- 操作

// Installed はパッケージが導入済みかを返す。
// script で入れるものは check コマンドが PATH にあるかで判定する。
func Installed(pm Manager, p Package) bool {
	if p.ViaScript() {
		return run.Look(p.Check)
	}
	if p.ID == "" {
		return false
	}
	return pm.IsInstalled(p.ID)
}

// Forget は Installed のキャッシュを捨てる。
func Forget(pm Manager, p Package) {
	if p.ID != "" {
		pm.Forget(p.ID)
	}
}

// Install はパッケージを導入する。
func Install(pm Manager, p Package, rep run.Reporter) error {
	if p.ViaScript() {
		rep.Info("[%s] script: %s", p.Name, p.Script)
		return runScript(p.Script, rep)
	}
	rep.Info("[%s] %s install %s", p.Name, pm.Name(), p.ID)
	return pm.Install(p.ID, rep)
}

// Uninstall はパッケージを削除する。
// script で入れたものはパッケージマネージャの管理外なので削除できない。
func Uninstall(pm Manager, p Package, rep run.Reporter) error {
	if p.ViaScript() {
		return fmt.Errorf("%s は script で導入したものなので削除できません", p.Name)
	}
	rep.Info("[%s] %s uninstall %s", p.Name, pm.Name(), p.ID)
	return pm.Uninstall(p.ID, rep)
}

// runScript は script をその OS のシェルで実行する。
func runScript(script string, rep run.Reporter) error {
	if runtime.GOOS == "windows" {
		return run.Stream(rep, "pwsh", "-NoLogo", "-NoProfile", "-Command", script)
	}
	return run.Stream(rep, "sh", "-c", script)
}

// HostLabel は画面表示用の OS 名を返す。
func HostLabel() string {
	switch runtime.GOOS {
	case "darwin":
		return "macOS"
	case "windows":
		return "Windows"
	default:
		return "Linux"
	}
}
