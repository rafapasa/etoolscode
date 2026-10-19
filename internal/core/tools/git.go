package tools

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/rafapasa/etoolscode/internal/core"
)

// FileStatus representa uma linha de git status --porcelain.
type FileStatus struct {
	Path   string `json:"path"`
	Index  string `json:"index"`  // status no index (1º char)
	Work   string `json:"work"`   // status no working tree (2º char)
	Staged bool   `json:"staged"`
}

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

func isGitRepo(root string) bool {
	_, err := os.Stat(filepath.Join(root, ".git"))
	return err == nil
}

// GitStatusRaw retorna o porcelain completo.
func GitStatusRaw(root string) (string, error) {
	if !isGitRepo(root) {
		return "", fmt.Errorf("nao e repo git: %s", root)
	}
	return gitRun(root, root, "status", "--porcelain", "-b")
}

// GitStatus lista arquivos modificados (sem a linha de branch).
func GitStatus(root string) ([]FileStatus, error) {
	if !isGitRepo(root) {
		return nil, fmt.Errorf("nao e repo git: %s", root)
	}
	out, err := gitRun(root, root, "status", "--porcelain")
	if err != nil {
		return nil, err
	}
	var list []FileStatus
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimRight(line, "\r")
		if len(line) < 4 {
			continue
		}
		idx := string(line[0])
		work := string(line[1])
		path := strings.TrimSpace(line[3:])
		// rename: "R  old -> new"
		if strings.Contains(path, " -> ") {
			parts := strings.Split(path, " -> ")
			path = strings.TrimSpace(parts[len(parts)-1])
		}
		list = append(list, FileStatus{
			Path:   path,
			Index:  idx,
			Work:   work,
			Staged: idx != " " && idx != "?",
		})
	}
	return list, nil
}

func GitDiff(root string, staged bool) (string, error) {
	if !isGitRepo(root) {
		return "", fmt.Errorf("nao e repo git: %s", root)
	}
	if staged {
		return gitRun(root, root, "diff", "--staged")
	}
	return gitRun(root, root, "diff")
}

// GitDiffFile diff de um único arquivo (working tree).
func GitDiffFile(root, file string) (string, error) {
	if !isGitRepo(root) {
		return "", fmt.Errorf("nao e repo git: %s", root)
	}
	file = filepath.Clean(file)
	if strings.HasPrefix(file, "..") || filepath.IsAbs(file) {
		return "", fmt.Errorf("path de arquivo invalido")
	}
	return gitRun(root, root, "diff", "--", file)
}

func GitCurrentBranch(root string) (string, error) {
	out, err := gitRun(root, root, "branch", "--show-current")
	return strings.TrimSpace(out), err
}

// GitCreateBranch cria branch e faz checkout (alias estável do plano).
func GitCreateBranch(root, name string) error {
	_, err := GitBranchCreate(root, name)
	return err
}

func GitBranchCreate(root, name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("nome branch vazio")
	}
	name = sanitizeBranch(name)
	if name == "" {
		return "", fmt.Errorf("nome branch invalido")
	}
	out, err := gitRun(root, root, "checkout", "-b", name)
	if err != nil && strings.Contains(out, "already exists") {
		out2, err2 := gitRun(root, root, "checkout", name)
		return out2, err2
	}
	return out, err
}

// GitCheckout troca para uma branch existente.
func GitCheckout(root, branch string) error {
	branch = sanitizeBranch(branch)
	if branch == "" {
		return fmt.Errorf("branch vazia")
	}
	out, err := gitRun(root, root, "checkout", branch)
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(out))
	}
	return nil
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
		message = fmt.Sprintf("etoolscode: auto commit %s", time.Now().Format("2006-01-02 15:04"))
	}
	return gitRun(root, root, "commit", "-m", message)
}

// EnsureBranch cria branch dsac/<issue>-<ts> se houver .git; se não houver, retorna skip silencioso.
func EnsureBranch(root, issueID string) (string, string, error) {
	if !isGitRepo(root) {
		return "", "", nil // skip sem erro
	}
	branchName := fmt.Sprintf("dsac/%s-%d", sanitizeBranch(issueID), time.Now().Unix()%100000)
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
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, ".json", "")
	var b strings.Builder
	prevSep := false
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			b.WriteRune(r)
			prevSep = false
			continue
		}
		if r == '/' {
			if b.Len() > 0 && !prevSep {
				b.WriteRune('/')
				prevSep = true
			}
			continue
		}
		if r == '-' || r == ' ' || r == '.' {
			if b.Len() > 0 && !prevSep {
				b.WriteRune('-')
				prevSep = true
			}
		}
	}
	res := strings.Trim(b.String(), "-/")
	if len(res) > 60 {
		res = res[:60]
	}
	return res
}
