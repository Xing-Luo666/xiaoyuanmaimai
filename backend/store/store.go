package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type JSONStore struct {
	mu   sync.RWMutex
	dir  string
}

func NewJSONStore(dir string) *JSONStore {
	os.MkdirAll(dir, 0755)
	return &JSONStore{dir: dir}
}

func (s *JSONStore) Read(filename string, v interface{}) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	path := filepath.Join(s.dir, filename)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(data, v)
}

func (s *JSONStore) Write(filename string, v interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := filepath.Join(s.dir, filename)
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (s *JSONStore) Exists(filename string) bool {
	path := filepath.Join(s.dir, filename)
	_, err := os.Stat(path)
	return err == nil
}
