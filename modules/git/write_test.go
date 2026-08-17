package git

import (
	"strings"
	"testing"
)

func TestFilterWriteCommitSuccess(t *testing.T) {
	raw := "[main abc1234] feat: thing\n 1 file changed, 2 insertions(+)\n"
	out := filterWrite("commit", raw, 0)
	if out != "ok abc1234\n" {
		t.Fatalf("got %q", out)
	}
}

func TestFilterWriteCommitFailurePassthrough(t *testing.T) {
	raw := "nothing to commit, working tree clean\n"
	out := filterWrite("commit", raw, 1)
	if out != raw {
		t.Fatalf("failed commit must pass through, got %q", out)
	}
}

func TestFilterWriteAddSuccess(t *testing.T) {
	out := filterWrite("add", "", 0)
	if out != "ok\n" {
		t.Fatalf("got %q", out)
	}
}

func TestFilterStashListCap(t *testing.T) {
	var sb strings.Builder
	for i := 0; i < 30; i++ {
		sb.WriteString("stash@{0}: WIP on main\n")
	}
	out := filterStashList(sb.String())
	if !strings.Contains(out, "+10 more") {
		t.Fatalf("got %q", out)
	}
}

func TestFilterBranchCap(t *testing.T) {
	var sb strings.Builder
	sb.WriteString("* main\n")
	for i := 0; i < 25; i++ {
		sb.WriteString("  branch\n")
	}
	out := filterBranch(sb.String())
	if !strings.Contains(out, "+10 more") {
		t.Fatalf("got %q", out)
	}
}

func TestRewritePushQuiet(t *testing.T) {
	m := &Module{}
	got, ok := m.Rewrite([]string{"push"})
	if !ok {
		t.Fatal("expected rewrite")
	}
	if len(got) != 2 || got[0] != "push" || got[1] != "-q" {
		t.Fatalf("got %v", got)
	}
}

func TestRewritePushVerboseSkipped(t *testing.T) {
	m := &Module{}
	_, ok := m.Rewrite([]string{"push", "-v"})
	if ok {
		t.Fatal("verbose push must not inject -q")
	}
}
