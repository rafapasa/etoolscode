package service

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
)

type FileIndex struct {
	Path        string         `json:"path"`
	Package     string         `json:"package"`
	Symbols     []string       `json:"symbols"`
	Imports     []string       `json:"imports"`
	ContentHash string         `json:"content_hash"`
	Summary     string         `json:"summary"`
	TokenCount  int            `json:"token_count"`
	TermFreq    map[string]int `json:"-"`
}

type RAGService struct {
	mu       sync.RWMutex
	root     string
	files    map[string]FileIndex
	inverted map[string]map[string]int
	docFreq  map[string]int
}

var ignoredDirs = map[string]bool{
	".git": true, "node_modules": true, "vendor": true, ".ds-ac": true,
	"dist": true, "build": true, ".next": true, ".idea": true, ".vscode": true,
}

var supportedExt = map[string]bool{
	".go": true, ".js": true, ".ts": true, ".jsx": true, ".tsx": true, ".dart": true,
	".json": true, ".md": true,
}

var (
	goPkgRe  = regexp.MustCompile(`(?m)^\s*package\s+(\w+)`)
	goFuncRe = regexp.MustCompile(`(?m)^\s*func\s+(?:\([^)]+\)\s+)?(\w+)`)
)

func NewRAGService(root string) *RAGService {
	return &RAGService{root: root, files: make(map[string]FileIndex), inverted: make(map[string]map[string]int), docFreq: make(map[string]int)}
}

func (r *RAGService) BuildIndex(root string) error {
	if root == "" {
		root = r.root
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.files = make(map[string]FileIndex)
	r.inverted = make(map[string]map[string]int)
	r.docFreq = make(map[string]int)
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if ignoredDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if !supportedExt[ext] {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		if len(data) > 500*1024 {
			return nil
		}
		content := string(data)
		h := sha256.Sum256(data)
		rel, _ := filepath.Rel(root, path)
		fi := FileIndex{Path: filepath.ToSlash(rel), ContentHash: fmt.Sprintf("%x", h[:8]), TokenCount: len(content) / 4, TermFreq: make(map[string]int)}
		if ext == ".go" {
			if m := goPkgRe.FindStringSubmatch(content); len(m) > 1 {
				fi.Package = m[1]
			}
			for _, m := range goFuncRe.FindAllStringSubmatch(content, 50) {
				if len(m) > 1 {
					fi.Symbols = append(fi.Symbols, m[1])
				}
			}
		}
		lines := strings.Split(content, "\n")
		if len(lines) > 20 {
			lines = lines[:20]
		}
		fi.Summary = strings.Join(lines, "\n")
		if len(fi.Summary) > 1500 {
			fi.Summary = fi.Summary[:1500]
		}
		words := tokenize(content)
		for _, w := range words {
			fi.TermFreq[w]++
		}
		r.files[fi.Path] = fi
		seen := make(map[string]bool)
		for term, freq := range fi.TermFreq {
			if r.inverted[term] == nil {
				r.inverted[term] = make(map[string]int)
			}
			r.inverted[term][fi.Path] = freq
			if !seen[term] {
				r.docFreq[term]++
				seen[term] = true
			}
		}
		return nil
	})
}

func tokenize(s string) []string {
	s = strings.ToLower(s)
	var words []string
	var cur strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			cur.WriteRune(r)
		} else {
			if cur.Len() >= 2 {
				w := cur.String()
				if len(w) >= 2 && len(w) <= 30 {
					words = append(words, w)
				}
			}
			cur.Reset()
		}
	}
	if cur.Len() >= 2 {
		words = append(words, cur.String())
	}
	return words
}

type SearchResult struct {
	File  FileIndex
	Score float64
}

func (r *RAGService) Search(query string, limit int) []SearchResult {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if limit <= 0 {
		limit = 10
	}
	terms := tokenize(query)
	scores := make(map[string]float64)
	for _, term := range terms {
		docs, ok := r.inverted[term]
		if !ok {
			continue
		}
		idf := 1.0
		if df, ok := r.docFreq[term]; ok && df > 0 {
			idf = 1 + float64(len(r.files))/float64(df)
		}
		for path, tf := range docs {
			scores[path] += float64(tf) * idf
		}
	}
	var results []SearchResult
	for path, score := range scores {
		if fi, ok := r.files[path]; ok {
			lowerPath := strings.ToLower(path)
			for _, t := range terms {
				if strings.Contains(lowerPath, t) {
					score += 5
				}
			}
			results = append(results, SearchResult{File: fi, Score: score})
		}
	}
	sort.Slice(results, func(i, j int) bool { return results[i].Score > results[j].Score })
	if len(results) > limit {
		results = results[:limit]
	}
	return results
}

func (r *RAGService) GetRelevantContext(query string, maxTokens int) string {
	if maxTokens <= 0 {
		maxTokens = 3000
	}
	results := r.Search(query, 15)
	var sb strings.Builder
	used := 0
	for _, res := range results {
		if used >= maxTokens {
			break
		}
		block := fmt.Sprintf("\n--- FILE: %s (score %.1f) ---\nPackage: %s\nSymbols: %s\nSummary:\n%s\n", res.File.Path, res.Score, res.File.Package, strings.Join(res.File.Symbols, ", "), res.File.Summary)
		tokens := len(block) / 4
		if used+tokens > maxTokens {
			break
		}
		sb.WriteString(block)
		used += tokens
	}
	return sb.String()
}

func (r *RAGService) ListFiles() []FileIndex {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]FileIndex, 0, len(r.files))
	for _, f := range r.files {
		out = append(out, f)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}
