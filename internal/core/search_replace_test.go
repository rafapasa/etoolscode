package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestApplySearchReplace(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "sample.go")
	orig := "package x\n\nfunc A() {}\n\nfunc B() {}\n"
	if err := os.WriteFile(p, []byte(orig), 0644); err != nil {
		t.Fatal(err)
	}
	n, err := ApplySearchReplace(p, "func A() {}", "func A() int { return 1 }", false)
	if err != nil || n != 1 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	b, _ := os.ReadFile(p)
	if !contains(string(b), "func A() int") {
		t.Fatalf("content: %s", b)
	}
	_, err = ApplySearchReplace(p, "func ", "func X", false)
	if err == nil {
		t.Fatal("expected multiple match error")
	}
	n, err = ApplySearchReplace(p, "func ", "func ", true)
	if err != nil || n < 2 {
		t.Fatalf("replace_all n=%d err=%v", n, err)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}
func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
