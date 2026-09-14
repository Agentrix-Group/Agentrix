package replay

import (
	"encoding/json"
	"errors"

	"github.com/F4nk1/Agentrix/src/model"
)

func Serialize(data *model.ReplayData) ([]byte, error) {
	if data == nil {
		return nil, errors.New("replay data is nil")
	}
	return json.MarshalIndent(data, "", "  ")
}

func Deserialize(raw []byte) (*model.ReplayData, error) {
	var data model.ReplayData
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

func Validate(data *model.ReplayData) error {
	if data == nil {
		return errors.New("replay data is nil")
	}
	if data.MatchId == "" {
		return errors.New("missing match_id in replay")
	}
	if data.GameId == "" {
		return errors.New("missing game_id in replay")
	}
	if len(data.Frames) == 0 {
		return errors.New("replay contains no frames")
	}
	return nil
}
