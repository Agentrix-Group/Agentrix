package model

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCategoryJSONSerialization(t *testing.T) {
	r := require.New(t)

	cat := Category{
		Id:          "cat-1",
		Description: "Strategy",
		Active:      true,
	}

	data, err := json.Marshal(cat)
	r.NoError(err)
	r.Contains(string(data), `"id":"cat-1"`)
	r.Contains(string(data), `"description":"Strategy"`)

	var parsed Category
	err = json.Unmarshal(data, &parsed)
	r.NoError(err)
	r.Equal("cat-1", parsed.Id)
	r.Equal("Strategy", parsed.Description)
	r.True(parsed.Active)
}
