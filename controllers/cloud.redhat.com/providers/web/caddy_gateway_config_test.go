package web

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCaddyConfig(t *testing.T) {
	ff, err := os.ReadFile("caddy_gateway_config_test.json")

	assert.NoError(t, err)

	e, _ := GenerateConfig("host", "bop", []string{"wer"}, []ProxyRoute{{
		Upstream: "11",
		Path:     "22",
	}})
	fmt.Print(e)
	assert.Equal(t, string(ff), e)
}
