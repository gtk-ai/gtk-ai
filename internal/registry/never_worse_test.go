package registry

import (
	"strings"
	"testing"
)

func TestNeverWorse(t *testing.T) {
	raw := strings.Repeat("a", 400)
	if NeverWorse(raw, "ok") != "ok" {
		t.Fatal("shorter filtered should win")
	}
	if NeverWorse("{}", "{\n  \"pretty\": true\n}") != "{}" {
		t.Fatal("longer filtered should fall back to raw")
	}
	if NeverWorse("abcd", "wxyz") != "wxyz" {
		t.Fatal("tie keeps filtered")
	}
}
