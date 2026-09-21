// Package game loads game modules. A module is a directory games/<id>/ with a
// manifest.yaml that declares everything the generic platform needs: player
// bounds, exact tick rate, limits, runtimes, opaque engine configuration,
// schemas and reference agents. The platform never interprets game rules.
package game

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/Agentrix-Group/Agentrix/src/model"
	"gopkg.in/yaml.v3"
)

type Players struct {
	Min int `yaml:"min"`
	Max int `yaml:"max"`
}

type Limits struct {
	TurnTimeoutMs int `yaml:"turn_timeout_ms"`
	InitTimeoutMs int `yaml:"init_timeout_ms"`
	WallTimeMs    int `yaml:"wall_time_ms"`
	MemoryMB      int `yaml:"memory_mb"`
	CPUSeconds    int `yaml:"cpu_seconds"`
	MaxProcesses  int `yaml:"max_processes"`
	MaxOutputKB   int `yaml:"max_output_kb"`
	MaxFileSizeKB int `yaml:"max_file_size_kb"`
	MaxStderrKB   int `yaml:"max_stderr_kb"`
	MaxLineBytes  int `yaml:"max_line_bytes"`
}

type ReferenceAgent struct {
	ID   string `yaml:"id"`
	Path string `yaml:"path"`
}

type Manifest struct {
	ID             string         `yaml:"id"`
	Name           string         `yaml:"name"`
	Version        string         `yaml:"version"`
	Description    string         `yaml:"description"`
	EngineProtocol string         `yaml:"engine_protocol"`
	BotProtocol    string         `yaml:"bot_protocol"`
	ReplayFormat   string         `yaml:"replay_format"`
	Viewer         string         `yaml:"viewer"`
	Players        Players        `yaml:"players"`
	TickRate       model.TickRate `yaml:"tick_rate"`
	MaxTicks       int            `yaml:"max_ticks"`
	Runtimes       []string       `yaml:"runtimes"`
	Limits         Limits         `yaml:"limits"`
	Engine         struct {
		Binary string `yaml:"binary"`
	} `yaml:"engine"`
	Config  map[string]any `yaml:"config"`
	Schemas struct {
		Perception string `yaml:"perception"`
		Action     string `yaml:"action"`
	} `yaml:"schemas"`
	Admission struct {
		Perception string `yaml:"perception"`
	} `yaml:"admission"`
	ReferenceAgents []ReferenceAgent `yaml:"reference_agents"`
}

// Module is a validated manifest bound to its directory.
type Module struct {
	Manifest
	Dir                 string
	admissionPerception json.RawMessage
}

var (
	idPattern      = regexp.MustCompile(`^[a-z][a-z0-9_-]{1,31}$`)
	versionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)
)

// SupportedRuntimes are the bot runtimes the sandbox implements.
var SupportedRuntimes = []string{"python3"}

func (m *Module) validate() error {
	switch {
	case !idPattern.MatchString(m.ID):
		return fmt.Errorf("invalid game id %q", m.ID)
	case m.Name == "":
		return errors.New("name is required")
	case !versionPattern.MatchString(m.Version):
		return fmt.Errorf("version %q must be MAJOR.MINOR.PATCH", m.Version)
	case m.EngineProtocol == "" || m.BotProtocol == "" || m.ReplayFormat == "":
		return errors.New("engine_protocol, bot_protocol and replay_format are required")
	case m.Players.Min < 1 || m.Players.Max < m.Players.Min:
		return errors.New("players must satisfy 1 <= min <= max")
	case !m.TickRate.Valid():
		return errors.New("tick_rate must be a positive ratio")
	case m.MaxTicks <= 0:
		return errors.New("max_ticks must be positive")
	case m.Engine.Binary == "":
		return errors.New("engine.binary is required")
	case m.Config == nil:
		return errors.New("config is required (use {} for none)")
	case m.Admission.Perception == "":
		return errors.New("admission.perception is required")
	}
	l := m.Limits
	if l.TurnTimeoutMs <= 0 || l.InitTimeoutMs <= 0 || l.WallTimeMs <= 0 || l.MemoryMB <= 0 || l.CPUSeconds <= 0 ||
		l.MaxProcesses <= 0 || l.MaxOutputKB <= 0 || l.MaxFileSizeKB <= 0 || l.MaxStderrKB <= 0 || l.MaxLineBytes <= 0 {
		return errors.New("every limit must be positive")
	}
	if len(m.Runtimes) == 0 {
		return errors.New("at least one runtime is required")
	}
	for _, rt := range m.Runtimes {
		if !contains(SupportedRuntimes, rt) {
			return fmt.Errorf("unsupported runtime %q", rt)
		}
	}
	if _, err := model.ConfigHash(m.Config); err != nil {
		return fmt.Errorf("config is not JSON serializable: %w", err)
	}
	for _, ref := range m.ReferenceAgents {
		if ref.ID == "" || !strings.HasSuffix(ref.Path, ".py") {
			return fmt.Errorf("invalid reference agent %q", ref.ID)
		}
		if _, err := os.Stat(m.Path(ref.Path)); err != nil {
			return fmt.Errorf("reference agent %q: %w", ref.ID, err)
		}
	}
	return nil
}

// Path resolves a path declared in the manifest relative to the module.
func (m *Module) Path(rel string) string { return filepath.Join(m.Dir, filepath.FromSlash(rel)) }

// ValidateRoster checks that a match with n slots is playable.
func (m *Module) ValidateRoster(n int) error {
	if n < m.Players.Min || n > m.Players.Max {
		return model.Validation("invalid_roster", "%s requires between %d and %d players, got %d",
			m.ID, m.Players.Min, m.Players.Max, n)
	}
	return nil
}

// Config returns a deep copy of the engine configuration.
func (m *Module) ConfigCopy() map[string]any {
	raw, _ := json.Marshal(m.Config) // validated at load time
	var out map[string]any
	_ = json.Unmarshal(raw, &out)
	return out
}

func (m *Module) ExecutionLimits() model.ExecutionLimits {
	l := m.Limits
	return model.ExecutionLimits{
		MaxTicks: m.MaxTicks, TurnTimeoutMs: l.TurnTimeoutMs, InitTimeoutMs: l.InitTimeoutMs,
		WallTimeMs: l.WallTimeMs, MemoryMB: l.MemoryMB, CPUSeconds: l.CPUSeconds, MaxProcesses: l.MaxProcesses,
		MaxOutputKB: l.MaxOutputKB, MaxFileSizeKB: l.MaxFileSizeKB, MaxStderrKB: l.MaxStderrKB, MaxLineBytes: l.MaxLineBytes,
	}
}

func (m *Module) AdmissionPerception() json.RawMessage {
	return append(json.RawMessage(nil), m.admissionPerception...)
}

// LoadModule reads and validates games/<id>/manifest.yaml strictly.
func LoadModule(dir string) (*Module, error) {
	raw, err := os.ReadFile(filepath.Join(dir, "manifest.yaml"))
	if err != nil {
		return nil, err
	}
	var manifest Manifest
	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	decoder.KnownFields(true)
	if err := decoder.Decode(&manifest); err != nil {
		return nil, fmt.Errorf("manifest %s: %w", dir, err)
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	module := &Module{Manifest: manifest, Dir: abs}
	if filepath.Base(abs) != manifest.ID {
		return nil, fmt.Errorf("manifest %s: directory name must equal id %q", dir, manifest.ID)
	}
	if err := module.validate(); err != nil {
		return nil, fmt.Errorf("manifest %s: %w", dir, err)
	}
	perception, err := os.ReadFile(module.Path(manifest.Admission.Perception))
	if err != nil {
		return nil, fmt.Errorf("manifest %s: admission perception: %w", dir, err)
	}
	if !json.Valid(perception) {
		return nil, fmt.Errorf("manifest %s: admission perception is not valid JSON", dir)
	}
	module.admissionPerception = bytes.TrimSpace(perception)
	return module, nil
}

// Registry holds the loaded game modules.
type Registry struct {
	mu      sync.RWMutex
	modules map[string]*Module
}

func NewRegistry(modules ...*Module) *Registry {
	r := &Registry{modules: map[string]*Module{}}
	for _, m := range modules {
		r.modules[m.ID] = m
	}
	return r
}

// LoadRegistry loads every games/*/manifest.yaml under root.
func LoadRegistry(root string) (*Registry, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("read games directory: %w", err)
	}
	registry := NewRegistry()
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(root, entry.Name(), "manifest.yaml")); err != nil {
			continue
		}
		module, err := LoadModule(filepath.Join(root, entry.Name()))
		if err != nil {
			return nil, err
		}
		registry.modules[module.ID] = module
	}
	if len(registry.modules) == 0 {
		return nil, fmt.Errorf("no game modules found under %s", root)
	}
	return registry, nil
}

func (r *Registry) Get(id string) (*Module, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	m, ok := r.modules[id]
	return m, ok
}

func (r *Registry) List() []*Module {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*Module, 0, len(r.modules))
	for _, m := range r.modules {
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func contains(list []string, v string) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}
