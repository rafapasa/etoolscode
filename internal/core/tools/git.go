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
	Index  string `json:"index"`
	Work   string `json:"work"`
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

func GitStatusRaw(root string) (string, error) {
	if !isGitRepo(root) {
		return "", fmt.Errorf("nao e repo git: %s", root)
	}
	return gitRun(root, root, "status", "--porcelain", "-b")
}

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

func GitDiffFile(root, file string) (string, error) {
	if !isGitRepo(root) {
		return "", fmt.Errorf("nao e repo git: %s", root)
	}
	file = cleanRel(file)
	if file == "" {
		return "", fmt.Errorf("path de arquivo invalido")
	}
	return gitRun(root, root, "diff", "--", file)
}

// GitFileAtHEAD retorna o conteúdo do arquivo no HEAD (vazio se não existir).
func GitFileAtHEAD(root, file string) (string, error) {
	if !isGitRepo(root) {
		return "", fmt.Errorf("nao e repo git: %s", root)
	}
	file = cleanRel(file)
	if file == "" {
		return "", fmt.Errorf("path de arquivo invalido")
	}
	out, err := gitRun(root, root, "show", "HEAD:"+filepath.ToSlash(file))
	if err != nil {
		// arquivo novo / inexistente no HEAD
		if strings.Contains(out, "does not exist") || strings.Contains(out, "exists on disk") || strings.Contains(out, "fatal") {
			return "", nil
		}
		return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(out))
	}
	return out, nil
}

// GitFileWorking lê o arquivo do working tree.
func GitFileWorking(root, file string) (string, error) {
	file = cleanRel(file)
	if file == "" {
		return "", fmt.Errorf("path de arquivo invalido")
	}
	full, err := core.ResolveSafePath(root, file)
	if err != nil {
		return "", err
	}
	b, err := os.ReadFile(full)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func cleanRel(file string) string {
	file = filepath.Clean(strings.TrimSpace(file))
	if file == "." || file == ".." || strings.HasPrefix(file, "..") || filepath.IsAbs(file) {
		return ""
	}
	return filepath.ToSlash(file)
}

func GitCurrentBranch(root string) (string, error) {
	out, err := gitRun(root, root, "branch", "--show-current")
	return strings.TrimSpace(out), err
}

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

// EnsureBranch cria/checkout dsac/<project>/<issue> se houver .git.
// project e issue vazios geram fallback com timestamp.
func EnsureBranch(root, project, issueID string) (string, string, error) {
	return EnsureBranchNamed(root, "", project, issueID)
}

// EnsureBranchNamed cria/checkout da branch.
// Se explicit != "", usa esse nome (sanitizado), ex: "dev-123".
// Senão: dsac/<projeto>/<issue>.
func EnsureBranchNamed(root, explicit, project, issueID string) (string, string, error) {
	if !isGitRepo(root) {
		return "", "", nil
	}
	var branchName string
	if strings.TrimSpace(explicit) != "" {
		branchName = sanitizeBranchFull(explicit)
	} else {
		p := sanitizeBranch(project)
		iss := sanitizeBranch(issueID)
		if p == "" {
			p = "proj"
		}
		if iss == "" {
			iss = fmt.Sprintf("task-%d", time.Now().Unix()%100000)
		}
		iss = strings.ReplaceAll(iss, "/", "-")
		branchName = fmt.Sprintf("dsac/%s/%s", p, iss)
	}
	if branchName == "" {
		return "", "", fmt.Errorf("nome de branch vazio")
	}
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

// sanitizeBranchFull permite barras (dev/123) mas limpa caracteres perigosos.
func sanitizeBranchFull(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, " ", "-")
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_' || r == '/' || r == '.':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	out := strings.Trim(b.String(), "-/")
	for strings.Contains(out, "--") {
		out = strings.ReplaceAll(out, "--", "-")
	}
	return out
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
