package exchange

import (
    "testing"
)

func TestCoinMarketCapClient(t *testing.T) {

    t.Skip("CoinMarketCap turns their api into private, need to investigate more.")

    var client = registry.getClient("coinMarketCap").(*coinMarketCapClient)

    t.Run("GetSymbolPrice", func(t *testing.T) {
        sp, err := client.GetSymbolPrice("bitcoin")

        if err != nil {
            t.Fatalf("Unexpected error: %v", err)
        }
        if sp.Price == "" {
            t.Fatalf("Get an empty price?")
        }
    })

    t.Run("GetUnexistSymbolPrice", func(t *testing.T) {
        _, err := client.GetSymbolPrice("bitcoin222")

        if err == nil {
            t.Fatalf("Should throws error 'id not found'")
        }
    })
}
