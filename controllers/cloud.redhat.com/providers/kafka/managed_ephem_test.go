package kafka

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHTTPClientCacheSet(_ *testing.T) {
	hcc := newHTTPClientCahce()
	client := http.Client{}
	hcc.Set("test", &client)
}

func TestHTTPClientCacheGet(t *testing.T) {
	hcc := newHTTPClientCahce()
	client := http.Client{}
	hcc.Set("test", &client)
	_, ok := hcc.Get("test")
	assert.True(t, ok)
	_, notOK := hcc.Get("not-found")
	assert.False(t, notOK)
}

func TestHTTPClientCacheRemove(_ *testing.T) {
	hcc := newHTTPClientCahce()
	client := http.Client{}
	hcc.Set("test", &client)
	hcc.Remove("test")
}
