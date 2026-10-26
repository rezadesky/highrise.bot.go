package highrisebot

import (
	"encoding/json"
	"errors"
	"os"
	"sync"
	"time"
)

// StorageAdapter defines the persistence interface.
type StorageAdapter interface {
	Get(key string) ([]byte, error)
	Set(key string, val []byte, ttl ...time.Duration) error
	Delete(key string) error
}

// MemoryStorage provides an in-memory thread-safe key-value store with TTL.
type MemoryStorage struct {
	mu   sync.RWMutex
	data map[string]storageEntry
}

type storageEntry struct {
	val       []byte
	expiresAt time.Time
}

// NewMemoryStorage creates an in-memory storage adapter.
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		data: make(map[string]storageEntry),
	}
}

func (m *MemoryStorage) Get(key string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	e, exists := m.data[key]
	if !exists {
		return nil, errors.New("key not found")
	}
	if !e.expiresAt.IsZero() && time.Now().After(e.expiresAt) {
		return nil, errors.New("key expired")
	}
	return e.val, nil
}

func (m *MemoryStorage) Set(key string, val []byte, ttl ...time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	e := storageEntry{val: val}
	if len(ttl) > 0 && ttl[0] > 0 {
		e.expiresAt = time.Now().Add(ttl[0])
	}
	m.data[key] = e
	return nil
}

func (m *MemoryStorage) Delete(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
	return nil
}

// JSONFileStorage provides atomic file persistence on disk.
type JSONFileStorage struct {
	path string
	mu   sync.RWMutex
	mem  *MemoryStorage
}

// NewJSONFileStorage creates a persistent JSON file storage adapter.
func NewJSONFileStorage(filePath string) (*JSONFileStorage, error) {
	js := &JSONFileStorage{
		path: filePath,
		mem:  NewMemoryStorage(),
	}

	if data, err := os.ReadFile(filePath); err == nil {
		var diskData map[string][]byte
		if err := json.Unmarshal(data, &diskData); err == nil {
			for k, v := range diskData {
				_ = js.mem.Set(k, v)
			}
		}
	}
	return js, nil
}

func (j *JSONFileStorage) Save() error {
	j.mu.Lock()
	defer j.mu.Unlock()

	j.mem.mu.RLock()
	diskMap := make(map[string][]byte, len(j.mem.data))
	for k, v := range j.mem.data {
		diskMap[k] = v.val
	}
	j.mem.mu.RUnlock()

	data, err := json.MarshalIndent(diskMap, "", "  ")
	if err != nil {
		return err
	}

	tmp := j.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, j.path)
}
