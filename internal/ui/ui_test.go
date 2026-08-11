package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// タブは下ボーダー付きの 2 行ブロック。文字列連結だと 2 つ目が罫線の行にずれる。
func TestTabsAlign(t *testing.T) {
	m := model{width: 80}
	out := m.tabs()

	if h := lipgloss.Height(out); h != 2 {
		t.Fatalf("tabs height = %d, want 2:\n%s", h, out)
	}
	first := strings.SplitN(out, "\n", 2)[0]
	for _, label := range []string{"Packages", "Configs"} {
		if !strings.Contains(first, label) {
			t.Errorf("first line should contain %q:\n%s", label, out)
		}
	}
}

func TestRenderDiffLineClassifies(t *testing.T) {
	cases := []struct {
		line string
		want string
	}{
		{"  +added", diffAddStyle.Render("  +added")},
		{"  -removed", diffDelStyle.Render("  -removed")},
		{"  +++ b/foo", subtleStyle.Render("  +++ b/foo")},
		{"  --- a/foo", subtleStyle.Render("  --- a/foo")},
		{"  @@ -1,2 +1,2 @@", diffHunkStyle.Render("  @@ -1,2 +1,2 @@")},
		{"  diff --git a/foo b/foo", diffFileStyle.Render("  diff --git a/foo b/foo")},
		{"   context", subtleStyle.Render("   context")},
	}
	for _, c := range cases {
		if got := renderDiffLine(c.line); got != c.want {
			t.Errorf("renderDiffLine(%q) = %q, want %q", c.line, got, c.want)
		}
	}
}
