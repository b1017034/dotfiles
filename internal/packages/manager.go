package packages

import (
	"runtime"
	"strings"
	"sync"

	"github.com/b1017034/dotfiles/internal/run"
)

// Manager は OS ごとのパッケージマネージャを抽象化する。
type Manager interface {
	Name() string
	Available() bool
	// 結果はキャッシュされる
	IsInstalled(id string) bool
	// キャッシュを捨てて再取得する
	Refresh()
	// 1 件だけキャッシュを捨てる
	Forget(id string)

	Install(id string, rep run.Reporter) error
	Uninstall(id string, rep run.Reporter) error
}

func NewManager() Manager {
	if runtime.GOOS == "windows" {
		return &winget{cache: map[string]bool{}}
	}
	return &brew{}
}

type brew struct {
	mu     sync.Mutex
	loaded bool
	set    map[string]bool
}

func (b *brew) Name() string    { return "brew" }
func (b *brew) Available() bool { return run.Look("brew") }

func (b *brew) Refresh() {
	b.mu.Lock()
	b.loaded = false
	b.set = nil
	b.mu.Unlock()
	b.load()
}

func (b *brew) Forget(string) { b.Refresh() }

func (b *brew) load() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.loaded {
		return
	}
	b.loaded = true
	b.set = map[string]bool{}

	if !run.Look("brew") {
		return
	}
	for _, kind := range []string{"--formula", "--cask"} {
		out, err := run.Output("brew", "list", kind, "-1")
		if err != nil {
			continue
		}
		for _, line := range strings.Fields(out) {
			b.set[line] = true
		}
	}
}

func (b *brew) IsInstalled(id string) bool {
	if id == "" {
		return false
	}
	b.load()
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.set[id]
}

func (b *brew) Install(id string, rep run.Reporter) error {
	return run.Stream(rep, "brew", "install", id)
}

func (b *brew) Uninstall(id string, rep run.Reporter) error {
	return run.Stream(rep, "brew", "uninstall", id)
}

type winget struct {
	mu    sync.Mutex
	cache map[string]bool
}

func (w *winget) Name() string    { return "winget" }
func (w *winget) Available() bool { return run.Look("winget") }

func (w *winget) Refresh() {
	w.mu.Lock()
	w.cache = map[string]bool{}
	w.mu.Unlock()
}

func (w *winget) Forget(id string) {
	w.mu.Lock()
	delete(w.cache, id)
	w.mu.Unlock()
}

// winget list を 1 件ずつ叩くので遅い。呼び出し元は非同期で実行する。
func (w *winget) IsInstalled(id string) bool {
	if id == "" {
		return false
	}

	w.mu.Lock()
	if v, ok := w.cache[id]; ok {
		w.mu.Unlock()
		return v
	}
	w.mu.Unlock()

	installed := false
	if run.Look("winget") {
		out, err := run.Command("winget", "list", "--id", id, "--exact",
			"--disable-interactivity", "--accept-source-agreements").CombinedOutput()
		// 未導入のときは非 0 で終了する
		installed = err == nil && strings.Contains(string(out), id)
	}

	w.mu.Lock()
	if w.cache == nil {
		w.cache = map[string]bool{}
	}
	w.cache[id] = installed
	w.mu.Unlock()

	return installed
}

func (w *winget) Install(id string, rep run.Reporter) error {
	return run.Stream(rep, "winget", "install", "--id", id, "--exact", "--silent",
		"--disable-interactivity", "--accept-source-agreements", "--accept-package-agreements")
}

func (w *winget) Uninstall(id string, rep run.Reporter) error {
	return run.Stream(rep, "winget", "uninstall", "--id", id, "--exact", "--silent",
		"--disable-interactivity")
}
