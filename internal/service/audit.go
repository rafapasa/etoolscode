package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

type AuditEntry struct {
	ID         string    `json:"id"`
	Timestamp  time.Time `json:"timestamp"`
	Project    string    `json:"project"`
	ChatID     string    `json:"chat_id,omitempty"`
	IssueID    string    `json:"issue_id,omitempty"`
	Tool       string    `json:"tool"`
	Path       string    `json:"path,omitempty"`
	Success    bool      `json:"success"`
	Error      string    `json:"error,omitempty"`
	DurationMs int64     `json:"duration_ms"`
}

type AuditService struct {
	mu   sync.Mutex
	path string
}

func NewAuditService() *AuditService {
	home, _ := os.UserHomeDir()
	p := filepath.Join(home, ".ds-ac", "audit.jsonl")
	_ = os.MkdirAll(filepath.Dir(p), 0755)
	return &AuditService{path: p}
}

func (a *AuditService) Log(entry AuditEntry) AuditEntry {
	a.mu.Lock()
	defer a.mu.Unlock()
	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}
	f, _ := os.OpenFile(a.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if f != nil {
		defer f.Close()
		b, _ := json.Marshal(entry)
		f.Write(append(b, '\n'))
	}
	return entry
}

func (a *AuditService) List(project string, limit int) []AuditEntry {
	a.mu.Lock()
	defer a.mu.Unlock()
	data, _ := os.ReadFile(a.path)
	var out []AuditEntry
	lines := splitLines(string(data))
	for i := len(lines) - 1; i >= 0; i-- {
		var e AuditEntry
		if json.Unmarshal([]byte(lines[i]), &e) != nil {
			continue
		}
		if project != "" && e.Project != project {
			continue
		}
		out = append(out, e)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}
