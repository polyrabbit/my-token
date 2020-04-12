package exchange

import (
	"testing"
	"time"
)

func TestOKExClient(t *testing.T) {

	var client = new(okexClient)

	t.Run("GetKlinePrice", func(t *testing.T) {
		_, err := client.GetKlinePrice("bTC_usdt", "60", time.Now().Add(-time.Hour), time.Now())

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
	})

	t.Run("GetKlinePrice of unknown symbol", func(t *testing.T) {
		_, err := client.GetKlinePrice("abcedfg", "60", time.Now().Add(-time.Hour), time.Now())

		if err == nil {
			t.Fatalf("Expecting error when fetching unknown price, but get nil")
		}
		t.Logf("Returned error is %v, expected?", err)
	})

	t.Run("GetSymbolPrice", func(t *testing.T) {
		sp, err := client.GetSymbolPrice("bTC_usdt")

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
