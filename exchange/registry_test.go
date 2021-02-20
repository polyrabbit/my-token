package exchange

import (
	"testing"

	"github.com/polyrabbit/my-token/config"
	"github.com/polyrabbit/my-token/http"
)

var registry *Registry

func init() {
	cfg := config.Parse()
	registry = NewRegistry(cfg, http.New(cfg))
}

func TestRegistry_getClient(t *testing.T) {

	t.Run("get non-exist client", func(t *testing.T) {
		var client = registry.getClient("Non-exist")
		if client != nil {
			t.Fatalf("Get a non-existing exchange should return nil")
		}
	})

	t.Run("get exist client", func(t *testing.T) {
		var client = registry.getClient("BINANce")
		if client == nil {
			t.Fatalf("BINANce client should exist")
		}
	})
}
