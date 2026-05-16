package store

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

// StartHotReload listens for SIGHUP and reloads the BIN CSV data atomically.
//
// Trigger a reload from the shell:
//
//	kill -HUP $(pgrep bin-engine)
//
// The swap is atomic: in-flight requests complete with the old snapshot;
// requests arriving after Swap() see the freshly loaded data.
// Errors are logged; the current snapshot is preserved on failure.
//
// Runs until done is closed (typically at graceful shutdown).
func (s *BinStore) StartHotReload(rulesPath, perfPath string, done <-chan struct{}) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP)

	go func() {
		defer signal.Stop(sigCh)
		for {
			select {
			case <-sigCh:
				log.Println("🔄 SIGHUP — reloading BIN CSV data...")
				if err := s.LoadFromCSV(rulesPath, perfPath); err != nil {
					log.Printf("❌ BIN reload failed: %v", err)
				} else {
					log.Printf("✅ BIN data reloaded (%d BIN prefixes)", s.RuleCount())
				}
			case <-done:
				return
			}
		}
	}()
}
