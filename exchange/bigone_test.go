package exchange

import (
	"testing"
)

func TestBigOneClient(t *testing.T) {

	t.Skip("No longer supports BigOne, as they don't have a stable api.")

	var client = &bigOneClient{}

	t.Run("GetSymbolPrice", func(t *testing.T) {
		sp, err := client.GetSymbolPrice("bTC-usdt")

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if sp.Price == "" {
			t.Fatalf("Get an empty price?")
		}
		if sp.PercentChange1h == 0 {
			t.Logf("WARNING - PercentChange1h is zero?")
		}
		if sp.PercentChange24h == 0 {
			t.Logf("WARNING - PercentChange24h is zero?")
		}
	})

	t.Run("GetUnexistSymbolPrice", func(t *testing.T) {
		_, err := client.GetSymbolPrice("ABC123")

		if err == nil {
			t.Fatalf("Should throws on invalid symbol")
		}
	})
}
