package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewService(t *testing.T) {
	r := require.New(t)

	svc := NewService(nil, nil, nil)
	r.NotNil(svc)
	s, ok := svc.(*service)
	r.True(ok)
	r.Nil(s.repo)
	r.Nil(s.artifacts)
	r.Nil(s.queue)
}
