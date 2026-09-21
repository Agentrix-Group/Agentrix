package server

import (
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/Agentrix-Group/Agentrix/src/config"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type openAPIDoc struct {
	Paths map[string]map[string]struct {
		Policy struct {
			Public        bool     `yaml:"public"`
			Capabilities  []string `yaml:"capabilities"`
			Authenticated bool     `yaml:"authenticated"`
		} `yaml:"x-agentrix-policy"`
	} `yaml:"paths"`
	Components struct {
		Schemas map[string]any `yaml:"schemas"`
	} `yaml:"components"`
}

// The OpenAPI document and the router must describe exactly the same
// operations with the same authorization policies.
func TestOpenAPIMatchesRouter(t *testing.T) {
	raw, err := os.ReadFile("../../open-api/openapi.yaml")
	require.NoError(t, err)
	var doc openAPIDoc
	require.NoError(t, yaml.Unmarshal(raw, &doc))

	s := New(nil, &config.Config{})
	defer s.Close()
	fromRouter := map[string]Route{}
	for _, r := range s.Routes() {
		fromRouter[strings.ToLower(r.Method)+" "+r.Pattern] = r
	}
	var fromDoc []string
	for path, ops := range doc.Paths {
		for method, op := range ops {
			key := method + " " + path
			fromDoc = append(fromDoc, key)
			route, ok := fromRouter[key]
			require.True(t, ok, "OpenAPI documents %s but the router does not serve it", key)
			require.Equal(t, route.Policy.Public, op.Policy.Public, "policy.public of %s", key)
			require.Equal(t, route.Policy.Authenticated, op.Policy.Authenticated, "policy.authenticated of %s", key)
			var caps []string
			for _, c := range route.Policy.Capabilities {
				caps = append(caps, string(c))
			}
			require.ElementsMatch(t, caps, op.Policy.Capabilities, "capabilities of %s", key)
		}
	}
	var missing []string
	for key := range fromRouter {
		found := false
		for _, d := range fromDoc {
			if d == key {
				found = true
			}
		}
		if !found {
			missing = append(missing, key)
		}
	}
	sort.Strings(missing)
	require.Empty(t, missing, "routes not documented in OpenAPI")

	refs := regexp.MustCompile(`#/components/schemas/([A-Za-z]+)`).FindAllStringSubmatch(string(raw), -1)
	for _, ref := range refs {
		_, ok := doc.Components.Schemas[ref[1]]
		require.True(t, ok, "unresolved schema reference %s", ref[1])
	}
	for _, legacy := range []string{"participant", "role_id", "code_path", "fixed_timestep_ms", "tick_hz", "/results", "pending", "completed"} {
		require.NotContains(t, string(raw), legacy, "legacy term %q in the contract", legacy)
	}
}

// DTO field names must equal the documented schema properties.
func TestOpenAPISchemasMatchDTOs(t *testing.T) {
	raw, err := os.ReadFile("../../open-api/openapi.yaml")
	require.NoError(t, err)
	var doc struct {
		Components struct {
			Schemas map[string]struct {
				Properties map[string]any `yaml:"properties"`
			} `yaml:"schemas"`
		} `yaml:"components"`
	}
	require.NoError(t, yaml.Unmarshal(raw, &doc))
	pairs := map[string]any{
		"User": UserDTO{}, "Session": SessionDTO{}, "Game": GameDTO{}, "Agent": AgentDTO{}, "Submission": SubmissionDTO{},
		"Contest": ContestDTO{}, "Entry": EntryDTO{}, "Slot": SlotDTO{}, "Match": MatchDTO{}, "Run": RunDTO{},
		"Result": ResultDTO{}, "Replay": ReplayDTO{}, "MatchDetail": MatchDetailDTO{}, "RankingRow": RankingDTO{},
		"Rankings": RankingsDTO{}, "Snapshot": SnapshotDTO{},
		"CreateAgentRequest": createAgentRequest{}, "CreateContestRequest": contestRequest{}, "EnrollRequest": enrollRequest{},
		"CreateMatchRequest": createMatchRequest{}, "CreateUserRequest": createUserRequest{}, "UpdateUserRequest": updateUserRequest{},
		"TransitionRequest": transitionRequest{}, "ReasonRequest": reasonRequest{}, "UserStatusRequest": userStatusRequest{},
		"AgentStatusRequest": agentStatusRequest{}, "RolesRequest": rolesRequest{}, "ResubmitRequest": resubmitRequest{},
	}
	for name, dto := range pairs {
		schema, ok := doc.Components.Schemas[name]
		require.True(t, ok, "schema %s missing", name)
		var documented []string
		for p := range schema.Properties {
			documented = append(documented, p)
		}
		require.ElementsMatch(t, jsonFields(reflect.TypeOf(dto)), documented, "schema %s vs DTO", name)
	}
}

func jsonFields(t reflect.Type) []string {
	var out []string
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.Anonymous {
			out = append(out, jsonFields(f.Type)...)
			continue
		}
		tag := strings.Split(f.Tag.Get("json"), ",")[0]
		if tag == "" || tag == "-" {
			continue
		}
		out = append(out, tag)
	}
	return out
}
