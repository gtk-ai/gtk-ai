package text

import "testing"

func TestStripANSI(t *testing.T) {
	in := "\x1b[32mgreen\x1b[0m file.go"
	got := StripANSI(in)
	if got != "green file.go" {
		t.Fatalf("got %q", got)
	}
}

func TestStripANSIPlain(t *testing.T) {
	in := "no color"
	if StripANSI(in) != in {
		t.Fatal("plain text should be unchanged")
	}
}
