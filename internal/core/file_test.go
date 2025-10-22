package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetHelloWorld(t *testing.T) {
	a := assert.New(t)

	text := GetHelloWorld()
	a.Equal("Hello World", text, "Should return 'Hello World'")
}
