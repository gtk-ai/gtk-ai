package shell

import (
	"testing"

	_ "github.com/jmeiracorbal/gtk-ai/modules/find"
	_ "github.com/jmeiracorbal/gtk-ai/modules/git"
	_ "github.com/jmeiracorbal/gtk-ai/modules/grep"
	_ "github.com/jmeiracorbal/gtk-ai/modules/ls"
	_ "github.com/jmeiracorbal/gtk-ai/modules/rg"
)

func TestRewriteGitStatus(t *testing.T) {
	got, ok := Rewrite("git status", "gtkai")
	if !ok || got != "gtkai git status" {
		t.Fatalf("got %q ok=%v", got, ok)
	}
}

func TestRewritePrefixes(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"/usr/bin/git status", "gtkai git status"},
		{"sudo git status", "sudo gtkai git status"},
		{"VAR=1 git status", "VAR=1 gtkai git status"},
		{"git -C /tmp status", "gtkai git -C /tmp status"},
		{"sudo /usr/bin/git -C /tmp status", "sudo gtkai git -C /tmp status"},
	}
	for _, tc := range cases {
		got, ok := Rewrite(tc.in, "gtkai")
		if !ok || got != tc.want {
			t.Errorf("%q: got %q ok=%v want %q", tc.in, got, ok, tc.want)
		}
	}
}

func TestRewriteUnregistered(t *testing.T) {
	_, ok := Rewrite("echo hi", "gtkai")
	if ok {
		t.Fatal("echo should not be rewritten")
	}
}

func TestRewriteAlreadyGtkai(t *testing.T) {
	_, ok := Rewrite("gtkai git status", "gtkai")
	if ok {
		t.Fatal("already-proxied command should not be rewritten")
	}
}

func TestRewriteEmptyBin(t *testing.T) {
	_, ok := Rewrite("git status", "")
	if ok {
		t.Fatal("empty gtkai path must not rewrite")
	}
}

func TestRewritePipelineLastGrep(t *testing.T) {
	got, ok := Rewrite("cat foo | grep bar", "gtkai")
	if !ok || got != "cat foo | gtkai grep bar" {
		t.Fatalf("got %q ok=%v", got, ok)
	}
}

func TestRewritePipelineGitPass(t *testing.T) {
	_, ok := Rewrite("git status | head", "gtkai")
	if ok {
		t.Fatal("unsafe pipeline last stage must pass through")
	}
}

func TestRewriteAnd(t *testing.T) {
	got, ok := Rewrite("cd /tmp && git status", "gtkai")
	if !ok || got != "cd /tmp && gtkai git status" {
		t.Fatalf("got %q ok=%v", got, ok)
	}
}

func TestRewriteRedirectPass(t *testing.T) {
	_, ok := Rewrite("git status > /tmp/out", "gtkai")
	if ok {
		t.Fatal("redirects must pass through")
	}
}

func TestRewriteRegisteredModules(t *testing.T) {
	cases := []struct{ in, want string }{
		{"ls", "gtkai ls"},
		{"ls -la /tmp", "gtkai ls -la /tmp"},
		{"find . -name '*.go'", "gtkai find . -name '*.go'"},
		{"grep -n foo src", "gtkai grep -n foo src"},
		{"rg Error src", "gtkai rg Error src"},
	}
	for _, tc := range cases {
		got, ok := Rewrite(tc.in, "gtkai")
		if !ok || got != tc.want {
			t.Errorf("%q: got %q ok=%v want %q", tc.in, got, ok, tc.want)
		}
	}
}
