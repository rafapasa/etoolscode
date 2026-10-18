package tools

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/rafapasa/etoolscode/internal/core"
)

func gitRun(root, cwd string, args ...string) (string, error) {
	if root == "" {
		root = cwd
	}
	if _, err := core.ResolveSafePath(root, "."); err != nil {
		return "", fmt.Errorf("git root invalido: %w", err)
	}
	cmd := exec.Command("git", args...)
	cmd.Dir = cwd
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func GitStatus(root string) (string, error) {
	if _, err := os.Stat(filepath.Join(root, ".git")); err != nil {
		return "", fmt.Errorf("nao e repo git: %s", root)
	}
	return gitRun(root, root, "status", "--porcelain", "-b")
}

func GitDiff(root string, staged bool) (string, error) {
	if staged {
		return gitRun(root, root, "diff", "--staged")
	}
	return gitRun(root, root, "diff")
}

func GitCurrentBranch(root string) (string, error) {
	out, err := gitRun(root, root, "branch", "--show-current")
	return strings.TrimSpace(out), err
}

func GitBranchCreate(root, name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("nome branch vazio")
	}
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "..", "")
	out, err := gitRun(root, root, "checkout", "-b", name)
	if err != nil && strings.Contains(out, "already exists") {
		out2, err2 := gitRun(root, root, "checkout", name)
		return out2, err2
	}
	return out, err
}

func GitCommit(root, message string, files []string) (string, error) {
	if len(files) > 0 {
		for _, f := range files {
			if _, err := gitRun(root, root, "add", f); err != nil {
				return "", err
			}
		}
	} else {
		if _, err := gitRun(root, root, "add", "-A"); err != nil {
			return "", err
		}
	}
	if message == "" {
		message = fmt.Sprintf("etoolcode: auto commit %s", time.Now().Format("2006-01-02 15:04"))
	}
	return gitRun(root, root, "commit", "-m", message)
}

func EnsureBranch(root, issueID string) (string, string, error) {
	branchName := fmt.Sprintf("etoolcode/%s-%d", sanitizeBranch(issueID), time.Now().Unix()%10000)
	cur, err := GitCurrentBranch(root)
	if err != nil {
		return "", "", err
	}
	if cur == branchName {
		return cur, branchName, nil
	}
	out, err := GitBranchCreate(root, branchName)
	return out, branchName, err
}

func sanitizeBranch(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, ".json", "")
	s = strings.ReplaceAll(s, "issue", "")
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		} else if r == ' ' {
			b.WriteRune('-')
		}
	}
	res := b.String()
	if len(res) > 30 {
		res = res[:30]
	}
	if res == "" {
		res = "task"
	}
	return res
}
