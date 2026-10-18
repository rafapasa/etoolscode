package chat

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound  = errors.New("chat: nao encontrado")
	ErrInvalidID = errors.New("chat: id invalido")
)

type ChatRepository interface {
	Save(session *ChatSession) error
	Load(id string) (*ChatSession, error)
	List(projectFilter string) ([]*ChatSession, error)
	Delete(id string) error
	ListByProject(project string) ([]*ChatSession, error)
}

type FileSystemRepository struct {
	mu  sync.RWMutex
	dir string
}

func NewFileSystemRepository(dir string) *FileSystemRepository {
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".ds-ac", "chats")
	}
	_ = os.MkdirAll(dir, 0755)
	return &FileSystemRepository{dir: dir}
}

func NewChatRepository() ChatRepository {
	return NewFileSystemRepository("")
}

func (r *FileSystemRepository) filePath(id string) (string, error) {
	if id == "" {
		return "", ErrInvalidID
	}
	for _, c := range id {
		if c == '.' || c == '/' {
			return "", ErrInvalidID
		}
	}
	return filepath.Join(r.dir, id+".json"), nil
}

func (r *FileSystemRepository) Save(session *ChatSession) error {
	if session == nil {
		return fmt.Errorf("session nil")
	}
	if session.ID == "" {
		session.ID = uuid.New().String()
	}
	now := time.Now()
	if session.CreatedAt.IsZero() {
		session.CreatedAt = now
	}
	session.UpdatedAt = now
	r.mu.Lock()
	defer r.mu.Unlock()
	path, err := r.filePath(session.ID)
	if err != nil {
		return err
	}
	data, _ := json.MarshalIndent(session, "", "  ")
	tmp := path + ".tmp"
	os.WriteFile(tmp, data, 0644)
	return os.Rename(tmp, path)
}

func (r *FileSystemRepository) Load(id string) (*ChatSession, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	path, err := r.filePath(id)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	var s ChatSession
	json.Unmarshal(data, &s)
	return &s, nil
}

func (r *FileSystemRepository) List(projectFilter string) ([]*ChatSession, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	entries, err := os.ReadDir(r.dir)
	if err != nil {
		return []*ChatSession{}, nil
	}
	var out []*ChatSession
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(r.dir, e.Name()))
		if err != nil {
			continue
		}
		var s ChatSession
		if json.Unmarshal(data, &s) != nil {
			continue
		}
		if projectFilter != "" && s.Project != projectFilter {
			continue
		}
		sc := s
		out = append(out, &sc)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UpdatedAt.After(out[j].UpdatedAt) })
	return out, nil
}

func (r *FileSystemRepository) ListByProject(project string) ([]*ChatSession, error) {
	return r.List(project)
}

func (r *FileSystemRepository) Delete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	path, err := r.filePath(id)
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if os.IsNotExist(err) {
		return ErrNotFound
	}
	return err
}
