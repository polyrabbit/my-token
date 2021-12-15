package exchange

import (
    "testing"
    "time"
)

func TestBittrexClient(t *testing.T) {

    var client = registry.getClient("bittrex").(*bittrexClient)

    t.Run("GetKlineTicks", func(t *testing.T) {
        klineResp, err := client.GetKlineTicks("USDT-BTc", "thirtyMin")

        if err != nil {
            t.Fatalf("Unexpected error: %v", err)
        }

        now := time.Now()
        lastHour := now.Add(-1 * time.Hour)
        _, err = client.GetPriceRightAfter(klineResp, lastHour)

        if err != nil {
            t.Fatalf("%s - Failed to get price 1 hour ago, error: %v\n", client.GetName(), err)
        }
    })

    t.Run("GetKlineTicks of unknown symbol", func(t *testing.T) {
        time.Sleep(time.Second * 1)
        _, err := client.GetKlineTicks("abcedfg", "thirtyMin")

        if err == nil {
            t.Fatalf("Expecting error when fetching unknown price, but get nil")
        }
        t.Logf("Returned error is %v, expected?", err)
    })

    t.Run("GetSymbolPrice", func(t *testing.T) {
        sp, err := client.GetSymbolPrice("USDT-BTc")

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
