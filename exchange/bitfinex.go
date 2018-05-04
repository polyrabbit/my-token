package exchange

import (
	"fmt"
	"github.com/bitfinexcom/bitfinex-api-go/v1"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// https://docs.bitfinex.com/docs

type bitfinixClient struct {
	innerClient *bitfinex.Client
}

func NewBitfinixClient(httpClient *http.Client) *bitfinixClient {
	http.DefaultClient = httpClient // luckily bitfinex uses the DefaultClient, override it here
	client := bitfinex.NewClient()
	return &bitfinixClient{innerClient: client}
}

func (client *bitfinixClient) GetName() string {
	return "Bitfinex"
}

func (client *bitfinixClient) GetPriceAt(symbol string, timestamp time.Time) (float64, error) {
	trades, err := client.innerClient.Trades.All(symbol, timestamp, 1)
	if err != nil {
		return 0, err
	}
	if len(trades) != 1 {
		return 0, fmt.Errorf("expecting only 1 trade returned, get %d", len(trades))
	}
	price, err := strconv.ParseFloat(trades[0].Price, 64)
	if err != nil {
		return 0, nil
	}
	return price, nil
}

func (client *bitfinixClient) GetSymbolPrice(symbol string) (*SymbolPrice, error) {
	ticker, err := client.innerClient.Ticker.Get(symbol)
	if err != nil {
		return nil, err
	}

	t, e := ticker.ParseTime()
	if e != nil {
		return nil, e
	}

	var percentChange1h, percentChange24h float64
	//Need api v2 to get kline
	//if currentPrice, err := strconv.ParseFloat(ticker.LastPrice, 64); err == nil {
	//	now := time.Now()
	//
	//	lastHour := now.Add(-1 * time.Hour)
	//	lastHourPrice, err := client.GetPriceAt(symbol, lastHour)
	//	if err != nil {
	//		logrus.Warnf("Failed to get price 1 hour ago, error: %s\n", err)
	//	}
	//	fmt.Printf("%s get last hour price %v\n", symbol, lastHourPrice)
	//	if lastHourPrice != 0 {
	//		percentChange1h = (currentPrice - lastHourPrice) / lastHourPrice * 100
	//	}
	//
	//	lastDay := now.Add(-24 * time.Hour)
	//	lastDayPrice, err := client.GetPriceAt(symbol, lastDay)
	//	if err != nil {
	//		logrus.Warnf("Failed to get price 24 hour ago, error: %s\n", err)
	//	}
	//	fmt.Printf("%s get last day price %v\n", symbol, lastDayPrice)
	//	if lastDayPrice != 0 {
	//		percentChange24h = (currentPrice - lastDayPrice) / lastDayPrice * 100
	//	}
	//} else {
	//	logrus.Warnf("Failed to convert current price %v to float", ticker.LastPrice)
	//}

	return &SymbolPrice{
		Symbol:           strings.ToUpper(symbol),
		Price:            ticker.LastPrice,
		Source:           client.GetName(),
		UpdateAt:         *t,
		PercentChange1h:  percentChange1h,
		PercentChange24h: percentChange24h,
	}, nil
}

func init() {
	register((&bitfinixClient{}).GetName(), func(client *http.Client) ExchangeClient {
		// Limited by type system in Go, I hate wrapper/adapter
		return NewBitfinixClient(client)
	})
}
