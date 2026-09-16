package game

import (
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

// Registry manages the game manifests available in Agentrix.
// Game execution is handled by external engine processes (Rust/Bevy/Rapier or fake-engine for tests)
// via the engine.EngineClient interface, keeping the game registry purely declarative.
type Registry struct {
	mu        sync.RWMutex
	manifests map[string]*Manifest
}

var defaultRegistry *Registry
var once sync.Once

// GetRegistry returns the global singleton Registry.
func GetRegistry() *Registry {
	once.Do(func() {
		defaultRegistry = NewRegistry()
	})
	return defaultRegistry
}

// NewRegistry creates a new empty Registry instance.
func NewRegistry() *Registry {
	return &Registry{
		manifests: make(map[string]*Manifest),
	}
}

// RegisterManifest registers or updates a game manifest.
func (r *Registry) RegisterManifest(manifest *Manifest) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.manifests[manifest.ID] = manifest
}

// GetManifest retrieves a manifest by its game ID, or nil if not found.
func (r *Registry) GetManifest(gameID string) *Manifest {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.manifests[gameID]
}

// LoadGamesFromDir scans a directory for subdirectories containing manifest.yaml files.
func (r *Registry) LoadGamesFromDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			manifestPath := filepath.Join(dir, entry.Name(), "manifest.yaml")
			if _, err := os.Stat(manifestPath); err == nil {
				data, err := os.ReadFile(manifestPath)
				if err != nil {
					continue
				}

				var manifest Manifest
				if err := yaml.Unmarshal(data, &manifest); err != nil {
					continue
				}

				if manifest.ID == "" {
					manifest.ID = entry.Name()
				}
				r.RegisterManifest(&manifest)
			}
		}
	}
	return nil
}

// ListManifests returns a list of all registered game manifests.
func (r *Registry) ListManifests() []*Manifest {
	r.mu.RLock()
	defer r.mu.RUnlock()
	list := make([]*Manifest, 0, len(r.manifests))
	for _, m := range r.manifests {
		list = append(list, m)
	}
	return list
}
