package integration

import (
	"bytes"
	"net/url"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/engine"
	"github.com/google/uuid"
)

func enginePkgClient() engine.EngineClient { return engine.NewSubprocessClient() }

func engineStart(bin string) engine.StartConfig {
	return engine.StartConfig{BinaryPath: bin, HandshakeTimeout: 10 * time.Second}
}

func bytesReader(b []byte) *bytes.Reader { return bytes.NewReader(b) }

func mustURL(raw string) *url.URL {
	u, err := url.Parse(raw)
	if err != nil {
		panic(err)
	}
	return u
}

func newUUID() string { return uuid.NewString() }
