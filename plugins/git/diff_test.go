package git

import (
	"strings"
	"testing"
)

func TestFilterDiffStripsHeadersAndCapsHunk(t *testing.T) {
	var sb strings.Builder
	sb.WriteString("diff --git a/main.go b/main.go\n")
	sb.WriteString("index abc..def 100644\n")
	sb.WriteString("--- a/main.go\n")
	sb.WriteString("+++ b/main.go\n")
	sb.WriteString("@@ -1,3 +1,4 @@\n")
	for i := 0; i < 150; i++ {
		sb.WriteString("+added line\n")
	}
	raw := sb.String()
	out := filterDiff(raw)
	if strings.Contains(out, "index ") || strings.Contains(out, "+++ ") {
		t.Fatalf("headers leaked: %s", out)
	}
	if !strings.Contains(out, "main.go") {
		t.Fatalf("missing file: %s", out)
	}
	if !strings.Contains(out, "truncated") {
		t.Fatalf("expected hunk cap: %s", out)
	}
	if strings.Count(out, "+added line") > maxHunkLines {
		t.Fatalf("hunk not capped: %d", strings.Count(out, "+added line"))
	}
}
