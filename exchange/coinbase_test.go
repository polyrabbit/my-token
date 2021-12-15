package exchange

import (
    "math"
    "testing"
)

func TestCoinbaseClient(t *testing.T) {

    var client = registry.getClient("coinbase").(*coinbaseClient)

    t.Run("GetSymbolPrice", func(t *testing.T) {
        sp, err := client.GetSymbolPrice("BTC-USd")

        if err != nil {
            t.Fatalf("Unexpected error: %v", err)
        }
        if sp.Price == "" {
            t.Fatalf("Get an empty price?")
        }
        if sp.PercentChange1h == math.MaxFloat64 {
            t.Logf("WARNING - PercentChange1h unset?")
        }
        if sp.PercentChange24h == math.MaxFloat64 {
            t.Logf("WARNING - PercentChange24h unset?")
        }
    })

    t.Run("GetUnexistSymbolPrice", func(t *testing.T) {
        _, err := client.GetSymbolPrice("ABC123")

        if err == nil {
            t.Fatalf("Should throws on invalid symbol")
        }
    })
}
