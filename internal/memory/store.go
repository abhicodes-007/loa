package memory

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/laughingmandev/loa/internal/state"
)

type Store struct {
	mu            sync.RWMutex
	path          string
	items         []state.MemoryItem
	summaries     []state.TaskSummary
	nextID        uint64
	halfLifeHours float64
	maxWeight     float64
}

func NewStore(root string, halfLifeHours float64, maxWeight float64) *Store {
	return NewStoreWithPath(filepath.Join(root, ".loa", "project_memory.json"), halfLifeHours, maxWeight)
}

func NewStoreWithPath(path string, halfLifeHours float64, maxWeight float64) *Store {
	return &Store{
		path:          path,
		nextID:        1,
		halfLifeHours: halfLifeHours,
		maxWeight:     maxWeight,
	}
}

func NewSessionStore(path string, halfLifeHours float64, maxWeight float64) *Store {
	return &Store{
		path:          path,
		nextID:        1,
		halfLifeHours: halfLifeHours,
		maxWeight:     maxWeight,
	}
}

func (s *Store) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var data struct {
		Items     []state.MemoryItem  `json:"items"`
		Summaries []state.TaskSummary `json:"summaries"`
		NextID    uint64              `json:"next_id"`
	}
	if err := json.Unmarshal(b, &data); err != nil {
		return err
	}
	s.items = data.Items
	s.summaries = data.Summaries
	s.nextID = data.NextID
	if s.nextID == 0 {
		s.nextID = 1
	}
	return nil
}

func (s *Store) saveLocked() error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	data := struct {
		Items     []state.MemoryItem  `json:"items"`
		Summaries []state.TaskSummary `json:"summaries"`
		NextID    uint64              `json:"next_id"`
	}{
		Items:     s.items,
		Summaries: s.summaries,
		NextID:    s.nextID,
	}
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return atomicWriteFile(s.path, b, 0600)
}

func atomicWriteFile(path string, b []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, "mem-tmp-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	ok := false
	defer func() {
		_ = f.Close()
		if !ok {
			_ = os.Remove(tmp)
		}
	}()
	if err = f.Chmod(mode); err != nil {
		return err
	}
	if _, err = f.Write(b); err != nil {
		return err
	}
	if err = f.Sync(); err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err = os.Rename(tmp, path); err != nil {
		return err
	}
	ok = true
	return nil
}

func (s *Store) NextID() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := s.nextID
	s.nextID++
	s.saveLocked()
	return id
}

func (s *Store) AddMemory(item state.MemoryItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if item.ID == 0 {
		item.ID = s.nextID
		s.nextID++
	}
	s.items = append(s.items, item)
	return s.saveLocked()
}

func (s *Store) UpdateMemory(id uint64, text string, emb []float32, indexKeys []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	for i, it := range s.items {
		if it.ID == id {
			s.items[i].Text = text
			s.items[i].Embedding = emb
			if indexKeys != nil {
				s.items[i].IndexKeys = indexKeys
			}
			return s.saveLocked()
		}
	}
	return nil
}

func (s *Store) DeleteMemory(id uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	var kept []state.MemoryItem
	changed := false
	for _, it := range s.items {
		if it.ID == id {
			changed = true
		} else {
			kept = append(kept, it)
		}
	}
	if changed {
		s.items = kept
		return s.saveLocked()
	}
	return nil
}

func (s *Store) AddSummary(summary state.TaskSummary) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.summaries = append(s.summaries, summary)
	return s.saveLocked()
}

func (s *Store) Search(emb []float32, limit int, now time.Time, targets []string, allowedKinds []state.MemoryKind) []Candidate {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return Search(s.items, s.summaries, emb, limit, now, targets, s.halfLifeHours, s.maxWeight, allowedKinds)
}

func (s *Store) Items() []state.MemoryItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]state.MemoryItem, len(s.items))
	copy(out, s.items)
	return out
}

func (s *Store) Summaries() []state.TaskSummary {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]state.TaskSummary, len(s.summaries))
	copy(out, s.summaries)
	return out
}

func (s *Store) ClearFileMemories() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var kept []state.MemoryItem
	for _, it := range s.items {
		if it.Kind != state.MemoryStructural && len(it.Anchors) == 0 {
			kept = append(kept, it)
		}
	}
	s.items = kept
	return s.saveLocked()
}

func (s *Store) RemoveMemoriesByAnchor(anchor string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var kept []state.MemoryItem
	changed := false
	for _, it := range s.items {
		match := false
		for _, a := range it.Anchors {
			if a == anchor {
				match = true
				break
			}
		}
		if match {
			changed = true
		} else {
			kept = append(kept, it)
		}
	}
	if changed {
		s.items = kept
		return s.saveLocked()
	}
	return nil
}

func (s *Store) FileMemories(path string) []state.MemoryItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []state.MemoryItem
	for _, it := range s.items {
		if it.Kind == state.MemoryStructural {
			for _, a := range it.Anchors {
				if a == path {
					out = append(out, it)
					break
				}
			}
		}
	}
	return out
}

func (s *Store) GlobalMemories() []state.MemoryItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []state.MemoryItem
	for _, it := range s.items {
		if it.Kind == state.MemoryGlobalConcept {
			out = append(out, it)
		}
	}
	return out
}

func (s *Store) SessionMemories() []state.MemoryItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []state.MemoryItem
	for _, it := range s.items {
		if it.Kind != state.MemoryStructural {
			out = append(out, it)
		}
	}
	return out
}
