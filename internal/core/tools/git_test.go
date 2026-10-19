package tools

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func initTempRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v (%s)", args, err, out)
		}
	}
	run("init")
	run("config", "user.email", "t@t")
	run("config", "user.name", "t")
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello\n"), 0644); err != nil {
		t.Fatal(err)
	}
	run("add", "a.txt")
	run("commit", "-m", "init")
	return dir
}

func TestGitStatusAndDiff(t *testing.T) {
	dir := initTempRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello world\n"), 0644); err != nil {
		t.Fatal(err)
	}
	st, err := GitStatus(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(st) == 0 {
		t.Fatal("esperava arquivo modificado")
	}
	diff, err := GitDiff(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(diff, "hello world") {
		t.Fatalf("diff sem conteudo: %s", diff)
	}
}

func TestGitCreateBranchAndCheckout(t *testing.T) {
	dir := initTempRepo(t)
	if err := GitCreateBranch(dir, "dsac/test-branch"); err != nil {
		t.Fatal(err)
	}
	cur, err := GitCurrentBranch(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cur != "dsac/test-branch" {
		t.Fatalf("branch atual=%s", cur)
	}
	if err := GitCheckout(dir, "main"); err != nil {
		// algumas versões usam master
		if err2 := GitCheckout(dir, "master"); err2 != nil {
			t.Fatalf("checkout main/master: %v / %v", err, err2)
		}
	}
}

func TestGitStatusNonRepo(t *testing.T) {
	dir := t.TempDir()
	_, err := GitStatus(dir)
	if err == nil {
		t.Fatal("esperava erro fora de repo")
	}
}

func TestEnsureBranchSkipNonGit(t *testing.T) {
	dir := t.TempDir()
	_, name, err := EnsureBranch(dir, "issue-1")
	if err != nil {
		t.Fatal(err)
	}
	if name != "" {
		t.Fatalf("esperava skip, got %s", name)
	}
}
