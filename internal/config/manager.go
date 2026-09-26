package config

import "sync"

type Manager struct {
	mu sync.RWMutex
	c  Config
}

func NewManager(c Config) *Manager { return &Manager{c: c} }
func (m *Manager) Get() Config     { m.mu.RLock(); defer m.mu.RUnlock(); return clone(m.c) }
func (m *Manager) Set(c Config)    { c.normalize(); m.mu.Lock(); m.c = clone(c); m.mu.Unlock() }
func clone(c Config) Config {
	out := c
	out.Permissions.AskFor = mapClone(c.Permissions.AskFor)
	return out
}
func mapClone[K comparable, V any](in map[K]V) map[K]V {
	if in == nil {
		return nil
	}
	out := make(map[K]V, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
