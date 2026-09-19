package executor

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type agentRef struct {
	ID   string `yaml:"id"`
	Path string `yaml:"path"`
}

type manifestDoc struct {
	ID              string     `yaml:"id"`
	ReferenceAgents []agentRef `yaml:"reference_agents"`
	Agents          struct {
		Reference []agentRef `yaml:"reference"`
	} `yaml:"agents"`
}

// ResolveAgentFallback dynamically resolves a reference agent from manifest.yaml for the given slot.
func ResolveAgentFallback(gameDir string, slot int) (string, error) {
	manifestPath := filepath.Join(gameDir, "manifest.yaml")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		if altData, altErr := os.ReadFile(filepath.Join("../..", manifestPath)); altErr == nil {
			data = altData
			gameDir = filepath.Join("../..", gameDir)
		} else {
			return "", fmt.Errorf("no se pudo leer manifest: %w", err)
		}
	}

	var m manifestDoc
	if err := yaml.Unmarshal(data, &m); err != nil {
		return "", fmt.Errorf("manifest corrupto: %w", err)
	}

	refs := m.ReferenceAgents
	if len(refs) == 0 && len(m.Agents.Reference) > 0 {
		refs = m.Agents.Reference
	}
	if len(refs) == 0 {
		return "", fmt.Errorf("el juego %s no declara agentes de referencia", m.ID)
	}

	if slot < 0 {
		slot = -slot
	}
	chosen := refs[slot%len(refs)]

	// Check direct path, path relative to gameDir, and path relative from subpackage tests
	candidates := []string{
		chosen.Path,
		filepath.Join(gameDir, chosen.Path),
		filepath.Join("../..", chosen.Path),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	return chosen.Path, fmt.Errorf("reference agent does not exist: %s", chosen.Path)
}
