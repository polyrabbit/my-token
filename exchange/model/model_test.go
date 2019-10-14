package model

import (
	"testing"
)

func TestTypes(t *testing.T) {

	t.Run("Exchange client factory", func(t *testing.T) {
		var client = getExchangeClient("Non-exist")
		if client != nil {
			t.Fatalf("Creating a non-existing exchange should return nil")
		}
	})
}
