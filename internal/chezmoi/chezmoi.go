// Package chezmoi は chezmoi コマンドの薄いラッパー。
// 設定ファイルの状態管理は全て chezmoi に委譲し、ここでは呼び出しと出力の整形だけを行う。
package chezmoi

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/b1017034/dotfiles/internal/run"
)

// ErrNotFound は chezmoi が PATH に無いことを表す。
var ErrNotFound = errors.New("chezmoi が見つかりません")

// Client は chezmoi コマンドを呼ぶ。
type Client struct{}

// New は chezmoi が使える場合に Client を返す。
func New() (*Client, error) {
	if !run.Look("chezmoi") {
		return nil, ErrNotFound
	}
	return &Client{}, nil
}

// SourceDir は chezmoi の source dir (= <repo>/home) を返す。
func (c *Client) SourceDir() (string, error) {
	out, err := run.Output("chezmoi", "source-path")
	if err != nil {
		return "", fmt.Errorf("chezmoi source-path に失敗しました: %w", err)
	}
	dir := strings.TrimSpace(out)
	if dir == "" {
		return "", errors.New("chezmoi source-path が空を返しました")
	}
	return dir, nil
}

// ---------------------------------------------------------------- status

// Entry は chezmoi status の 1 行。
type Entry struct {
	// Local は 1 文字目。chezmoi が最後に書いた状態と実際の差、
	// つまり chezmoi を通さずターゲット側を直接いじったかどうか。
	Local byte
	// Target は 2 文字目。実際の状態とソースが期待する状態の差、
	// つまり apply で何が起きるか。
	Target byte
	// Path はターゲット側の絶対パス。
	Path string
}

// Code は "MM" のような 2 文字のコードを返す。
func (e Entry) Code() string { return string([]byte{e.Local, e.Target}) }

// Dirty は apply で何か起きる状態かを返す。
func (e Entry) Dirty() bool { return e.Target != ' ' && e.Target != 0 }

// LocallyModified は chezmoi を通さずターゲット側が変更されたかを返す。
// この状態で apply すると chezmoi が上書き確認のプロンプトを出すため、
// 呼び出し側が先に確認を取って Apply の force を立てる必要がある。
func (e Entry) LocallyModified() bool {
	switch e.Local {
	case 'A', 'M', 'D':
		return true
	}
	return false
}

// Label はコードを日本語の一言に直す。
func (e Entry) Label() string {
	switch e.Target {
	case 'A':
		return "未作成"
	case 'M':
		if e.LocallyModified() {
			return "差分あり（ターゲット側も変更済み）"
		}
		return "差分あり"
	case 'D':
		return "削除予定"
	case 'R':
		return "スクリプト実行予定"
	}
	if e.LocallyModified() {
		return "ターゲット側が変更済み"
	}
	return "一致"
}

// Status は chezmoi status の結果を返す。差分が無ければ空スライス。
func (c *Client) Status() ([]Entry, error) {
	out, err := run.Output("chezmoi", "status", "--path-style=absolute")
	if err != nil {
		return nil, fmt.Errorf("chezmoi status に失敗しました: %w", err)
	}

	var entries []Entry
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimRight(line, "\r")
		if len(line) < 3 {
			continue
		}
		path := strings.TrimSpace(line[2:])
		if path == "" {
			continue
		}
		entries = append(entries, Entry{Local: line[0], Target: line[1], Path: path})
	}
	return entries, nil
}

// LocallyModified は entries のうちターゲット側が変更済みのパスを返す。
// paths が空でなければ、その配下にあるものだけを見る。
// 戻り値が空でなければ Apply の前にユーザーの確認が要る。
func LocallyModified(entries []Entry, paths []string) []string {
	var out []string
	for _, e := range entries {
		if !e.LocallyModified() {
			continue
		}
		if len(paths) > 0 && !covers(paths, e.Path) {
			continue
		}
		out = append(out, e.Path)
	}
	return out
}

// covers は paths のいずれかが target 自身か、その祖先ディレクトリかを返す。
// chezmoi はディレクトリを指定すると配下を再帰的に扱うので、それに合わせる。
func covers(paths []string, target string) bool {
	for _, p := range paths {
		abs, err := filepath.Abs(expandHome(p))
		if err != nil {
			abs = p
		}
		if abs == target {
			return true
		}
		rel, err := filepath.Rel(abs, target)
		if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

// expandHome は先頭の ~ をホームディレクトリに展開する。
// シェルが展開しないまま渡されたときのため。
func expandHome(p string) string {
	if p != "~" && !strings.HasPrefix(p, "~/") {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return p
	}
	return filepath.Join(home, strings.TrimPrefix(p, "~"))
}

// Managed は chezmoi が管理しているターゲットの絶対パス一覧を返す。
func (c *Client) Managed() ([]string, error) {
	out, err := run.Output("chezmoi", "managed", "--path-style=absolute")
	if err != nil {
		return nil, fmt.Errorf("chezmoi managed に失敗しました: %w", err)
	}
	var paths []string
	for _, line := range strings.Split(out, "\n") {
		if p := strings.TrimSpace(line); p != "" {
			paths = append(paths, p)
		}
	}
	return paths, nil
}

// ---------------------------------------------------------------- 操作

// Diff は差分を Reporter に流す。
func (c *Client) Diff(rep run.Reporter, paths ...string) error {
	rep.Info("chezmoi diff %s", strings.Join(short(paths), " "))
	err := run.Stream(rep, "chezmoi", append([]string{"diff", "--no-pager"}, paths...)...)
	if err != nil {
		rep.Err("差分の取得に失敗しました: %v", err)
	}
	return err
}

// Apply は差分を適用する。paths が空なら全体を対象にする。
//
// force はターゲット側の変更を確認せず上書きすることを表す。
// chezmoi は「chezmoi が最後に書いた後にターゲットが変わっている」場合に
// 上書き確認のプロンプトを出すが、ここは TTY を持たないので答えられない。
// LocallyModified なエントリがあるかは呼び出し側が Status で判断し、
// ユーザーの確認を取った上で force を立てる。
func (c *Client) Apply(rep run.Reporter, force bool, paths ...string) error {
	args := []string{"apply", "-v", "--no-tty"}
	if force {
		args = append(args, "--force")
	}
	rep.Info("chezmoi apply %s", strings.Join(short(paths), " "))
	if err := run.Stream(rep, "chezmoi", append(args, paths...)...); err != nil {
		rep.Err("適用に失敗しました: %v", err)
		return err
	}
	rep.OK("適用しました")
	return nil
}

// ReAdd はターゲット側の変更をソースに取り込む。
// 呼び出し側で確認を取ってから使う前提なので、プロンプトは出さない。
func (c *Client) ReAdd(rep run.Reporter, paths ...string) error {
	args := []string{"re-add", "-v", "--no-tty", "--force"}
	rep.Info("chezmoi re-add %s", strings.Join(short(paths), " "))
	if err := run.Stream(rep, "chezmoi", append(args, paths...)...); err != nil {
		rep.Err("取り込みに失敗しました: %v", err)
		return err
	}
	rep.OK("ソースに取り込みました")
	return nil
}

// short は表示用にホームディレクトリを ~ に縮める。
func short(paths []string) []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return paths
	}
	out := make([]string, len(paths))
	for i, p := range paths {
		if rel, err := filepath.Rel(home, p); err == nil && !strings.HasPrefix(rel, "..") {
			out[i] = filepath.Join("~", rel)
			continue
		}
		out[i] = p
	}
	return out
}

// Short は 1 件分のパスを表示用に縮める。
func Short(path string) string {
	return short([]string{path})[0]
}
