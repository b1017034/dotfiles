package ui

import "testing"

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
