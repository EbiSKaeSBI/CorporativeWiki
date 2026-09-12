package service_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheck(t *testing.T) {
	svc := newTestService(t)

	assert.Equal(t, "ok", svc.Check())
}