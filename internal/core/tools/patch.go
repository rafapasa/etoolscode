package tools

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var hunkHeaderRegex = regexp.MustCompile(`^@@ -(\d+),?(\d*) \+(\d+),?(\d*) @@`)

type hunk struct {
	oldStart int
	oldCount int
	newStart int
	newCount int
	lines    []string
}

func parseHunk(line string) (hunk, bool) {
	m := hunkHeaderRegex.FindStringSubmatch(line)
	if m == nil {
		return hunk{}, false
	}
	oldStart, _ := strconv.Atoi(m[1])
	oldCount := 1
	if m[2] != "" {
		oldCount, _ = strconv.Atoi(m[2])
	}
	newStart, _ := strconv.Atoi(m[3])
	newCount := 1
	if m[4] != "" {
		newCount, _ = strconv.Atoi(m[4])
	}
	return hunk{oldStart: oldStart, oldCount: oldCount, newStart: newStart, newCount: newCount}, true
}

func ApplyPatch(filePath, diffContent string) (string, error) {
	cleanPath := filepath.Clean(filePath)
	if cleanPath == "" {
		return "", fmt.Errorf("filePath vazio")
	}
	trimmed := strings.TrimSpace(diffContent)
	if trimmed == "" {
		return "", fmt.Errorf("diff vazio")
	}

	if err := os.MkdirAll(filepath.Dir(cleanPath), 0755); err != nil {
		return "", fmt.Errorf("mkdir: %w", err)
	}

	if !strings.Contains(diffContent, "@@") {
		lines := strings.Split(diffContent, "\n")
		var cleaned []string
		for _, l := range lines {
			if strings.HasPrefix(l, "+++") || strings.HasPrefix(l, "---") || strings.HasPrefix(l, "diff ") || strings.HasPrefix(l, "index ") {
				continue
			}
			if strings.HasPrefix(l, "+") {
				cleaned = append(cleaned, l[1:])
			} else {
				cleaned = append(cleaned, l)
			}
		}
		final := strings.Join(cleaned, "\n")
		if strings.TrimSpace(final) == "" {
			final = diffContent
		}
		if err := os.WriteFile(cleanPath, []byte(final), 0644); err != nil {
			return "", err
		}
		return fmt.Sprintf("Arquivo criado/sobrescrito: %s (%d bytes, conteudo novo)", cleanPath, len(final)), nil
	}

	var original []string
	if data, err := os.ReadFile(cleanPath); err == nil {
		original = strings.Split(string(data), "\n")
	} else {
		original = []string{}
	}

	var hunks []hunk
	var cur *hunk
	scanner := bufio.NewScanner(strings.NewReader(diffContent))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "@@") {
			if h, ok := parseHunk(line); ok {
				if cur != nil {
					hunks = append(hunks, *cur)
				}
				cur = &h
			}
			continue
		}
		if cur == nil {
			continue
		}
		if strings.HasPrefix(line, "---") || strings.HasPrefix(line, "+++") {
			continue
		}
		if len(line) == 0 {
			cur.lines = append(cur.lines, " "+line)
		} else if line[0] == '+' || line[0] == '-' || line[0] == ' ' {
			cur.lines = append(cur.lines, line)
		} else if line[0] == '\\' {
			continue
		} else {
			cur.lines = append(cur.lines, " "+line)
		}
	}
	if cur != nil {
		hunks = append(hunks, *cur)
	}

	if len(hunks) == 0 {
		return "", fmt.Errorf("nenhum hunk valido encontrado")
	}

	result := make([]string, 0, len(original)+100)
	origIdx := 0

	for _, hk := range hunks {
		targetStart := hk.oldStart - 1
		if targetStart < 0 {
			targetStart = 0
		}
		for origIdx < targetStart && origIdx < len(original) {
			result = append(result, original[origIdx])
			origIdx++
		}
		for _, l := range hk.lines {
			if l == "" {
				continue
			}
			switch l[0] {
			case ' ':
				if origIdx < len(original) {
					result = append(result, original[origIdx])
					origIdx++
				} else {
					result = append(result, l[1:])
				}
			case '-':
				if origIdx < len(original) {
					origIdx++
				}
			case '+':
				result = append(result, l[1:])
			}
		}
	}
	for origIdx < len(original) {
		result = append(result, original[origIdx])
		origIdx++
	}

	finalContent := strings.Join(result, "\n")
	if err := os.WriteFile(cleanPath, []byte(finalContent), 0644); err != nil {
		return "", err
	}
	return fmt.Sprintf("Patch aplicado: %s (%d hunks, %d -> %d linhas)", cleanPath, len(hunks), len(original), len(result)), nil
}
