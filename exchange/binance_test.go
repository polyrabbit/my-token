package exchange

import (
	"testing"
)

func TestBinanceClient(t *testing.T) {

	var client = &binanceClient{}

	t.Run("Get24hStatistics", func(t *testing.T) {
		stat, err := client.Get24hStatistics("ethbtc")

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if stat.Msg != nil {
			t.Fatalf("Error message %s", *stat.Msg)
		}

		if stat.LastPrice == "" {
			t.Fatalf("Get unknown price")
		}
	})

	t.Run("GetPrice1hAgo", func(t *testing.T) {
		price, err := client.GetPrice1hAgo("ethbtc")

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if price == 0 {
			t.Fatalf("Price is zero? must be the end of word")
		}
	})

	t.Run("GetPrice1hAgo", func(t *testing.T) {
		price, err := client.GetPrice1hAgo("ethbtc")

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if price == 0 {
			t.Fatalf("Price is zero? must be the end of word")
		}
	})

	t.Run("GetSymbolPrice", func(t *testing.T) {
		sp, err := client.GetSymbolPrice("ethbtc")

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if sp.Price == "" {
			t.Fatalf("Get an empty price?")
		}
	})

	t.Run("GetUnexistSymbolPrice", func(t *testing.T) {
		_, err := client.GetSymbolPrice("ABC123")

		if err == nil {
			t.Fatalf("Should throws on invalid symbol")
		}
	})
}
