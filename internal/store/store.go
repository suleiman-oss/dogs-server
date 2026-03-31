package store

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
)

// Dogs is the in-memory representation: breed → list of sub-breeds.
type Dogs map[string][]string

// Store is a thread-safe, disk-backed dog registry.
type Store struct {
	mu       sync.RWMutex
	filePath string
	data     Dogs
}

// New loads data from filePath (or seeds from seedPath if filePath doesn't exist).
func New(filePath, seedPath string) (*Store, error) {
	s := &Store{filePath: filePath}

	// Use existing data file if present
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// Fall back to seed
		if seedPath != "" {
			if err := copyFile(seedPath, filePath); err != nil {
				return nil, fmt.Errorf("seeding data file: %w", err)
			}
		} else {
			// Start empty
			if err := s.flush(Dogs{}); err != nil {
				return nil, err
			}
		}
	}

	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

// ── Read ──────────────────────────────────────────────────────────────────────

// All returns a copy of every breed.
func (s *Store) All() Dogs {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(Dogs, len(s.data))
	for k, v := range s.data {
		cp := make([]string, len(v))
		copy(cp, v)
		out[k] = cp
	}
	return out
}

// Get returns the sub-breeds for a single breed, or an error if not found.
func (s *Store) Get(breed string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	subs, ok := s.data[breed]
	if !ok {
		return nil, fmt.Errorf("breed %q not found", breed)
	}
	cp := make([]string, len(subs))
	copy(cp, subs)
	return cp, nil
}

// ── Write ─────────────────────────────────────────────────────────────────────

// Create adds a new breed. Returns an error if it already exists.
func (s *Store) Create(breed string, subs []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[breed]; ok {
		return fmt.Errorf("breed %q already exists", breed)
	}
	if subs == nil {
		subs = []string{}
	}
	s.data[breed] = normaliseSubs(subs)
	return s.flush(s.data)
}

// Replace overwrites the sub-breeds list for an existing breed.
func (s *Store) Replace(breed string, subs []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[breed]; !ok {
		return fmt.Errorf("breed %q not found", breed)
	}
	if subs == nil {
		subs = []string{}
	}
	s.data[breed] = normaliseSubs(subs)
	return s.flush(s.data)
}

// AddSubs appends sub-breeds (deduplicates) to an existing breed.
func (s *Store) AddSubs(breed string, subs []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.data[breed]
	if !ok {
		return fmt.Errorf("breed %q not found", breed)
	}
	set := make(map[string]struct{}, len(existing))
	for _, v := range existing {
		set[v] = struct{}{}
	}
	for _, sub := range normaliseSubs(subs) {
		if _, dup := set[sub]; !dup {
			existing = append(existing, sub)
			set[sub] = struct{}{}
		}
	}
	s.data[breed] = existing
	return s.flush(s.data)
}

// DeleteBreed removes a breed entirely.
func (s *Store) DeleteBreed(breed string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[breed]; !ok {
		return fmt.Errorf("breed %q not found", breed)
	}
	delete(s.data, breed)
	return s.flush(s.data)
}

// DeleteSub removes one sub-breed from a breed.
func (s *Store) DeleteSub(breed, sub string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	subs, ok := s.data[breed]
	if !ok {
		return fmt.Errorf("breed %q not found", breed)
	}
	idx := -1
	for i, v := range subs {
		if v == sub {
			idx = i
			break
		}
	}
	if idx == -1 {
		return fmt.Errorf("sub-breed %q not found under %q", sub, breed)
	}
	s.data[breed] = append(subs[:idx], subs[idx+1:]...)
	return s.flush(s.data)
}

// ── Internals ─────────────────────────────────────────────────────────────────

func (s *Store) load() error {
	b, err := os.ReadFile(s.filePath)
	if err != nil {
		return fmt.Errorf("reading data file: %w", err)
	}
	var d Dogs
	if err := json.Unmarshal(b, &d); err != nil {
		return fmt.Errorf("parsing data file: %w", err)
	}
	s.data = d
	return nil
}

// flush writes data to disk in alphabetical key order (no lock – caller holds it).
func (s *Store) flush(d Dogs) error {
	// Sort keys
	keys := make([]string, 0, len(d))
	for k := range d {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	ordered := make(map[string][]string, len(d))
	for _, k := range keys {
		ordered[k] = d[k]
	}

	b, err := json.MarshalIndent(ordered, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.filePath, b, 0644)
}

func normaliseSubs(subs []string) []string {
	out := make([]string, 0, len(subs))
	for _, s := range subs {
		n := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(s), " ", ""))
		if n != "" {
			out = append(out, n)
		}
	}
	return out
}

func copyFile(src, dst string) error {
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, b, 0644)
}
