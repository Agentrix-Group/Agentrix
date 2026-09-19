package game

import (
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

// Registry exposes the single Starfighter manifest supported by the MVP.
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

// RegisterManifest registers or updates the Starfighter manifest.
func (r *Registry) RegisterManifest(manifest *Manifest) {
	if manifest == nil || manifest.ID != "starfighter" {
		return
	}
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

// LoadGamesFromDir loads only games/starfighter/manifest.yaml.
func (r *Registry) LoadGamesFromDir(dir string) error {
	manifestPath := filepath.Join(dir, "starfighter", "manifest.yaml")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return err
	}
	var manifest Manifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return err
	}
	if err := ValidateManifest(&manifest); err != nil {
		return err
	}
	r.RegisterManifest(&manifest)
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
