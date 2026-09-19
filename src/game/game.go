package game

type Manifest struct {
	ID              string            `yaml:"id" json:"id"`
	Name            string            `yaml:"name" json:"name"`
	Version         string            `yaml:"version" json:"version"`
	Description     string            `yaml:"description" json:"description"`
	MinPlayers      int               `yaml:"min_players" json:"min_players"`
	MaxPlayers      int               `yaml:"max_players" json:"max_players"`
	MaxTicks        int               `yaml:"max_ticks" json:"max_ticks"`
	BinaryPath      string            `yaml:"binary_path,omitempty" json:"binary_path,omitempty"`
	FixedTimestepMs int               `yaml:"fixed_timestep_ms,omitempty" json:"fixed_timestep_ms,omitempty"`
	ReferenceAgents []ReferenceAgent  `yaml:"reference_agents,omitempty" json:"reference_agents,omitempty"`
	Settings        map[string]string `yaml:"settings,omitempty" json:"settings,omitempty"`
}

type ReferenceAgent struct {
	ID   string `yaml:"id" json:"id"`
	Path string `yaml:"path" json:"path"`
}
