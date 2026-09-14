package game

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

type EngineFactory func(manifest *Manifest) Engine

type Registry struct {
	mu        sync.RWMutex
	manifests map[string]*Manifest
	factories map[string]EngineFactory
}

var defaultRegistry *Registry
var once sync.Once

func GetRegistry() *Registry {
	once.Do(func() {
		defaultRegistry = NewRegistry()
		defaultRegistry.RegisterFactory("arena-basica", NewArenaBasicaEngine)
	})
	return defaultRegistry
}

func NewRegistry() *Registry {
	return &Registry{
		manifests: make(map[string]*Manifest),
		factories: make(map[string]EngineFactory),
	}
}

func (r *Registry) RegisterFactory(gameID string, factory EngineFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.factories[gameID] = factory
}

func (r *Registry) RegisterManifest(manifest *Manifest) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.manifests[manifest.ID] = manifest
}

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

func (r *Registry) CreateEngine(gameID string) (Engine, error) {
	r.mu.RLock()
	factory, exists := r.factories[gameID]
	manifest := r.manifests[gameID]
	r.mu.RUnlock()

	if !exists {
		// Fallback to default arena-basica factory if recognized
		if gameID == "arena-basica" {
			factory = NewArenaBasicaEngine
		} else {
			return nil, fmt.Errorf("engine factory for game '%s' not found", gameID)
		}
	}

	if manifest == nil {
		manifest = &Manifest{
			ID:          gameID,
			Name:        "Arena Basica",
			Version:     "1.0.0",
			MinPlayers:  2,
			MaxPlayers:  4,
			MaxTicks:    100,
			GridWidth:   10,
			GridHeight:  10,
			Description: "Deterministic grid battle simulation",
		}
	}

	return factory(manifest), nil
}

func (r *Registry) ListManifests() []*Manifest {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var list []*Manifest
	for _, m := range r.manifests {
		list = append(list, m)
	}
	return list
}

// Arena Basica default Engine implementation
type arenaBasicaEngine struct {
	manifest *Manifest
	state    *GameState
	rng      *rand.Rand
}

func NewArenaBasicaEngine(manifest *Manifest) Engine {
	if manifest == nil {
		manifest = &Manifest{
			ID:         "arena-basica",
			Name:       "Arena Basica",
			Version:    "1.0.0",
			MinPlayers: 2,
			MaxPlayers: 4,
			MaxTicks:   100,
			GridWidth:  10,
			GridHeight: 10,
		}
	}
	return &arenaBasicaEngine{
		manifest: manifest,
	}
}

func (e *arenaBasicaEngine) GetManifest() *Manifest {
	return e.manifest
}

func (e *arenaBasicaEngine) Init(players []string, seed int64) (*GameState, error) {
	if len(players) < e.manifest.MinPlayers || len(players) > e.manifest.MaxPlayers {
		return nil, errors.New("invalid number of players for arena-basica")
	}

	e.rng = rand.New(rand.NewSource(seed))
	spawnPositions := [][2]int{
		{1, 1},
		{e.manifest.GridWidth - 2, e.manifest.GridHeight - 2},
		{1, e.manifest.GridHeight - 2},
		{e.manifest.GridWidth - 2, 1},
	}

	playerMap := make(map[string]*PlayerState)
	for i, pID := range players {
		pos := spawnPositions[i%len(spawnPositions)]
		playerMap[pID] = &PlayerState{
			ID:       pID,
			X:        pos[0],
			Y:        pos[1],
			HP:       100,
			MaxHP:    100,
			Energy:   100,
			Shielded: false,
			Alive:    true,
			Score:    0,
		}
	}

	e.state = &GameState{
		Tick:       0,
		GridWidth:  e.manifest.GridWidth,
		GridHeight: e.manifest.GridHeight,
		Players:    playerMap,
		Events:     []string{"Match started in Arena Basica"},
		Done:       false,
	}

	return e.state, nil
}

func (e *arenaBasicaEngine) Step(actions map[string]Action) (*GameState, error) {
	if e.state == nil {
		return nil, errors.New("game engine not initialized")
	}
	if e.state.Done {
		return e.state, nil
	}

	e.state.Tick++
	e.state.Events = []string{}

	// Reset shields
	for _, p := range e.state.Players {
		if p.Alive {
			p.Shielded = false
		}
	}

	// 1. Process Defense / Rest actions first
	for pID, act := range actions {
		p, ok := e.state.Players[pID]
		if !ok || !p.Alive {
			continue
		}

		switch act.Type {
		case ActionShield:
			if p.Energy >= 10 {
				p.Shielded = true
				p.Energy -= 10
				e.state.Events = append(e.state.Events, fmt.Sprintf("%s raised shield", pID))
			}
		case ActionRest:
			p.Energy = min(100, p.Energy+25)
			p.HP = min(p.MaxHP, p.HP+5)
			e.state.Events = append(e.state.Events, fmt.Sprintf("%s rested and recovered energy", pID))
		}
	}

	// 2. Process Movements
	for pID, act := range actions {
		p, ok := e.state.Players[pID]
		if !ok || !p.Alive {
			continue
		}

		newX, newY := p.X, p.Y
		switch act.Type {
		case ActionUp:
			newY = max(0, p.Y-1)
		case ActionDown:
			newY = min(e.manifest.GridHeight-1, p.Y+1)
		case ActionLeft:
			newX = max(0, p.X-1)
		case ActionRight:
			newX = min(e.manifest.GridWidth-1, p.X+1)
		default:
			continue
		}

		// Check collision with other alive players
		collision := false
		for otherID, other := range e.state.Players {
			if otherID != pID && other.Alive && other.X == newX && other.Y == newY {
				collision = true
				break
			}
		}

		if !collision {
			p.X = newX
			p.Y = newY
			e.state.Events = append(e.state.Events, fmt.Sprintf("%s moved to (%d,%d)", pID, p.X, p.Y))
		}
	}

	// 3. Process Attacks
	for pID, act := range actions {
		p, ok := e.state.Players[pID]
		if !ok || !p.Alive || act.Type != ActionAttack {
			continue
		}

		if p.Energy < 15 {
			e.state.Events = append(e.state.Events, fmt.Sprintf("%s attempted attack but lacked energy", pID))
			continue
		}
		p.Energy -= 15

		for otherID, other := range e.state.Players {
			if otherID == pID || !other.Alive {
				continue
			}

			// Adjacent attack (Manhattan distance <= 1)
			dist := abs(p.X-other.X) + abs(p.Y-other.Y)
			if dist <= 1 {
				dmg := 25
				if other.Shielded {
					dmg = 5
					e.state.Events = append(e.state.Events, fmt.Sprintf("%s shielded an attack from %s", otherID, pID))
				} else {
					e.state.Events = append(e.state.Events, fmt.Sprintf("%s struck %s for %d damage", pID, otherID, dmg))
				}

				other.HP -= dmg
				p.Score += dmg

				if other.HP <= 0 {
					other.HP = 0
					other.Alive = false
					p.Score += 50
					e.state.Events = append(e.state.Events, fmt.Sprintf("%s defeated %s!", pID, otherID))
				}
			}
		}
	}

	// Check end conditions
	aliveCount := 0
	lastAlive := ""
	for pID, p := range e.state.Players {
		if p.Alive {
			aliveCount++
			lastAlive = pID
		}
	}

	if aliveCount <= 1 || e.state.Tick >= e.manifest.MaxTicks {
		e.state.Done = true
		if aliveCount == 1 {
			e.state.Winner = lastAlive
			e.state.Players[lastAlive].Score += 100
			e.state.Events = append(e.state.Events, fmt.Sprintf("Match over! %s is the winner!", lastAlive))
		} else {
			e.state.Events = append(e.state.Events, "Match over! Draw or time limit reached.")
		}
	}

	return e.state, nil
}

func (e *arenaBasicaEngine) GetState() *GameState {
	return e.state
}

func (e *arenaBasicaEngine) IsOver() bool {
	return e.state != nil && e.state.Done
}

func (e *arenaBasicaEngine) GetResults() map[string]int {
	res := make(map[string]int)
	if e.state == nil {
		return res
	}
	for pID, p := range e.state.Players {
		res[pID] = p.Score
	}
	return res
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
