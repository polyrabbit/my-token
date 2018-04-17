package exchange

import (
	"net/http"
	"testing"
)

func TestBitfinixClient(t *testing.T) {

	var client = NewBitfinixClient(http.DefaultClient)

	t.Run("GetSymbolPrice", func(t *testing.T) {
		sp, err := client.GetSymbolPrice("btcusd")

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if sp.Price == "" {
			t.Fatalf("Get an empty price?")
		}
	})

	t.Run("GetUnexistSymbolPrice", func(t *testing.T) {
		_, err := client.GetSymbolPrice("btcusd121")

		if err == nil {
			t.Fatalf("Should throws error '400 Unknown symbol'")
		}
	})
}
