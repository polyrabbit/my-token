package exchange

import (
	"testing"
)

func TestTypes(t *testing.T) {

	t.Run("Exchange client factory", func(t *testing.T) {
		var client = CreateExchangeClient("Non-exist", nil)
		if client != nil {
			t.Fatalf("Creating a non-existing exchange should return nil")
		}
	})
}
