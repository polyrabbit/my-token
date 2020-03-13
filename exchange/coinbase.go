package exchange

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"time"

	"github.com/polyrabbit/my-token/http"

	"github.com/polyrabbit/my-token/exchange/model"

	"github.com/preichenberger/go-coinbasepro/v2"
	"github.com/sirupsen/logrus"
)

type coinbaseClient struct {
	coinbasepro *coinbasepro.Client
}

func NewCoinBaseClient() *coinbaseClient {
	client := coinbasepro.NewClient()
	return &coinbaseClient{coinbasepro: client}
}

func (client *coinbaseClient) Init() {
	client.coinbasepro.HTTPClient = http.HTTPClient
}

func (client *coinbaseClient) GetName() string {
	return "Coinbase"
}

func (client *coinbaseClient) GetPriceRightAfter(candles []coinbasepro.HistoricRate, after time.Time) (float64, error) {
	for _, candle := range candles {
		if after.Equal(candle.Time) || after.After(candle.Time) {
			// Assume candles are sorted in desc order, so the first less than or equal to is the candle looking for
			logrus.Debugf("%s - Kline for %v uses open price at %v", client.GetName(), after.Local(), candle.Time.Local())
			return candle.Open, nil
		}
	}
	return 0, fmt.Errorf("no time found right after %v", after)
}

func (client *coinbaseClient) GetSymbolPrice(symbol string) (*model.SymbolPrice, error) {
	ticker, err := client.coinbasepro.GetTicker(symbol)
	if err != nil {
		return nil, err
	}
	currentPrice, err := strconv.ParseFloat(ticker.Price, 64)
	if err != nil {
		return nil, err
	}

	var percentChange1h, percentChange24h = math.MaxFloat64, math.MaxFloat64
	candles, err := client.coinbasepro.GetHistoricRates(symbol, coinbasepro.GetHistoricRatesParams{
		Granularity: 300,
	})
	if err != nil {
		logrus.Warnf("%s - Failed to get kline ticks, error: %v", client.GetName(), err)
	} else {
		now := time.Now()
		sort.Slice(candles, func(i, j int) bool { return candles[i].Time.After(candles[j].Time) })

		lastHour := now.Add(-1 * time.Hour)
		price1hAgo, err := client.GetPriceRightAfter(candles, lastHour)
		if err != nil {
			logrus.Warnf("%s - Failed to get price 1 hour ago, error: %v\n", client.GetName(), err)
		} else if price1hAgo != 0 {
			percentChange1h = (currentPrice - price1hAgo) / price1hAgo * 100
		}

		last24Hour := now.Add(-24 * time.Hour)
		price24hAgo, err := client.GetPriceRightAfter(candles, last24Hour)
		if err != nil {
			logrus.Warnf("%s - Failed to get price 24 hours ago, error: %v\n", client.GetName(), err)
		} else if price24hAgo != 0 {
			percentChange24h = (currentPrice - price24hAgo) / price24hAgo * 100
		}
	}

	return &model.SymbolPrice{
		Symbol:           symbol,
		Price:            ticker.Price,
		UpdateAt:         time.Time(ticker.Time),
		Source:           client.GetName(),
		PercentChange1h:  percentChange1h,
		PercentChange24h: percentChange24h,
	}, nil
}

func init() {
	model.Register(NewCoinBaseClient())
}
