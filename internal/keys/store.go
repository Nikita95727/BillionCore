package keys

import (
	"encoding/json"
	"log"
	"os"
	"sync/atomic"
	"time"
)

// snapshot maps key_value → key_id (ULID).
// Replaced atomically on every reload; never mutated in place.
type snapshot map[string]string

// Store provides O(1), lock-free API-key validation backed by valid_keys.json
// that the Laravel control plane writes atomically whenever a key is
// created or revoked.
//
// Usage:
//
//	s := keys.NewStore("/tmp/billioncore_valid_keys.json")
//	s.Load()                               // once at startup
//	s.StartAutoReload(30*time.Second, done) // background refresh
//
//	keyID, ok := s.Check(keyValue)         // O(1), lock-free
type Store struct {
	ptr      atomic.Pointer[snapshot]
	filePath string
}

// NewStore creates a Store pointing at filePath.
// The snapshot is initially empty; call Load() before serving traffic.
func NewStore(filePath string) *Store {
	s := &Store{filePath: filePath}
	empty := make(snapshot)
	s.ptr.Store(&empty)
	return s
}

// Load reads filePath and atomically replaces the in-memory snapshot.
// Safe to call concurrently with Check().
func (s *Store) Load() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}
	var m snapshot
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	s.ptr.Store(&m)
	return nil
}

// Check returns the key_id for keyValue, or ("", false) if the key is unknown.
// O(1) and lock-free — one atomic pointer load followed by a map lookup.
func (s *Store) Check(keyValue string) (keyID string, ok bool) {
	m := *s.ptr.Load()
	keyID, ok = m[keyValue]
	return
}

// Len returns how many keys are currently loaded.
func (s *Store) Len() int {
	return len(*s.ptr.Load())
}

// StartAutoReload reloads the keys file every interval until done is closed.
// Reload errors are logged but do NOT replace the current snapshot.
func (s *Store) StartAutoReload(interval time.Duration, done <-chan struct{}) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := s.Load(); err != nil {
					log.Printf("⚠️  keys reload: %v", err)
				} else {
					log.Printf("🔑 keys reloaded (%d active)", s.Len())
				}
			case <-done:
				return
			}
		}
	}()
}
