package exchange

import (
    "testing"
    "time"
)

func TestZBClient(t *testing.T) {

    var client = registry.getClient("zb").(*zbClient)

    t.Run("GetKlinePrice", func(t *testing.T) {
        _, err := client.GetKlinePrice("bTC_usdt", "1min", 60)

        if err != nil {
            t.Fatalf("Unexpected error: %v", err)
        }
    })

    t.Run("GetKlinePrice of unknown symbol", func(t *testing.T) {
        time.Sleep(time.Second * 1)
        _, err := client.GetKlinePrice("abcedfg", "1min", 60)

        if err == nil {
            t.Fatalf("Expecting error when fetching unknown price, but get nil")
        }
        t.Logf("Returned error is `%v`, expected?", err)
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
