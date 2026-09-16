package game

type ActionType string

const (
	ActionUp     ActionType = "UP"
	ActionDown   ActionType = "DOWN"
	ActionLeft   ActionType = "LEFT"
	ActionRight  ActionType = "RIGHT"
	ActionAttack ActionType = "ATTACK"
	ActionShield ActionType = "SHIELD"
	ActionRest   ActionType = "REST"
)

type Action struct {
	PlayerID ActionType             `json:"-"`
	Type     ActionType             `json:"type"`
	Payload  map[string]interface{} `json:"payload,omitempty"`
}

type PlayerState struct {
	ID       string `json:"id"`
	X        int    `json:"x"`
	Y        int    `json:"y"`
	HP       int    `json:"hp"`
	MaxHP    int    `json:"max_hp"`
	Energy   int    `json:"energy"`
	Shielded bool   `json:"shielded"`
	Alive    bool   `json:"alive"`
	Score    int    `json:"score"`
}

type GameState struct {
	Tick       int                     `json:"tick"`
	GridWidth  int                     `json:"grid_width"`
	GridHeight int                     `json:"grid_height"`
	Players    map[string]*PlayerState `json:"players"`
	Events     []string                `json:"events"`
	Done       bool                    `json:"done"`
	Winner     string                  `json:"winner,omitempty"`
}

type Manifest struct {
	ID          string            `yaml:"id" json:"id"`
	Name        string            `yaml:"name" json:"name"`
	Version     string            `yaml:"version" json:"version"`
	Description string            `yaml:"description" json:"description"`
	MinPlayers  int               `yaml:"min_players" json:"min_players"`
	MaxPlayers  int               `yaml:"max_players" json:"max_players"`
	MaxTicks    int               `yaml:"max_ticks" json:"max_ticks"`
	GridWidth   int               `yaml:"grid_width" json:"grid_width"`
	GridHeight  int               `yaml:"grid_height" json:"grid_height"`
	BinaryPath  string            `yaml:"binary_path,omitempty" json:"binary_path,omitempty"`
	Settings    map[string]string `yaml:"settings,omitempty" json:"settings,omitempty"`
}
