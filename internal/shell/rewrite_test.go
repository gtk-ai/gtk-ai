package shell

import (
	"testing"

	_ "github.com/jmeiracorbal/gtk-ai/modules/cargo"
	_ "github.com/jmeiracorbal/gtk-ai/modules/find"
	_ "github.com/jmeiracorbal/gtk-ai/modules/go"
	_ "github.com/jmeiracorbal/gtk-ai/modules/git"
	_ "github.com/jmeiracorbal/gtk-ai/modules/grep"
	_ "github.com/jmeiracorbal/gtk-ai/modules/ls"
	_ "github.com/jmeiracorbal/gtk-ai/modules/docker"
	_ "github.com/jmeiracorbal/gtk-ai/modules/npmtest"
	_ "github.com/jmeiracorbal/gtk-ai/modules/pytest"
	_ "github.com/jmeiracorbal/gtk-ai/modules/python"
	_ "github.com/jmeiracorbal/gtk-ai/modules/readcmd"
	_ "github.com/jmeiracorbal/gtk-ai/modules/rg"
	_ "github.com/jmeiracorbal/gtk-ai/modules/tree"
)

func TestRewriteNpmTest(t *testing.T) {
	got, ok := Rewrite("npm test", "gtkai")
	if !ok || got != "gtkai npm test" {
		t.Fatalf("got %q ok=%v", got, ok)
	}
}

func TestRewriteDockerPS(t *testing.T) {
	got, ok := Rewrite("docker ps", "gtkai")
	if !ok || got != "gtkai docker ps" {
		t.Fatalf("got %q ok=%v", got, ok)
	}
}

func TestRewritePytest(t *testing.T) {
	got, ok := Rewrite("pytest -v", "gtkai")
	if !ok || got != "gtkai pytest -v" {
		t.Fatalf("got %q ok=%v", got, ok)
	}
}

func TestRewritePythonMPytest(t *testing.T) {
	got, ok := Rewrite("python -m pytest", "gtkai")
	if !ok || got != "gtkai python -m pytest" {
		t.Fatalf("got %q ok=%v", got, ok)
	}
	got, ok = Rewrite("python3 -m pytest tests/", "gtkai")
	if !ok || got != "gtkai python3 -m pytest tests/" {
		t.Fatalf("got %q ok=%v", got, ok)
	}
}

func TestRewriteCargoTest(t *testing.T) {
	got, ok := Rewrite("cargo test", "gtkai")
	if !ok || got != "gtkai cargo test" {
		t.Fatalf("got %q ok=%v", got, ok)
	}
}

func TestRewriteGoTest(t *testing.T) {
	got, ok := Rewrite("go test ./...", "gtkai")
	if !ok || got != "gtkai go test ./..." {
		t.Fatalf("got %q ok=%v", got, ok)
	}
}

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
		{"cat main.go", "gtkai cat main.go"},
		{"head -n 10 main.go", "gtkai head -n 10 main.go"},
		{"tree -L 2", "gtkai tree -L 2"},
	}
	for _, tc := range cases {
		got, ok := Rewrite(tc.in, "gtkai")
		if !ok || got != tc.want {
			t.Errorf("%q: got %q ok=%v want %q", tc.in, got, ok, tc.want)
		}
	}
}
