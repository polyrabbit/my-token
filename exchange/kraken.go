package exchange

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/polyrabbit/my-token/exchange/model"

	"github.com/polyrabbit/my-token/http"
	"github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

// https://www.kraken.com/help/api
const krakenBaseApi = "https://api.kraken.com/0/public/"

type krakenClient struct {
	*http.Client
	AccessKey string
	SecretKey string
}

func NewKrakenClient(httpClient *http.Client) ExchangeClient {
	return &krakenClient{Client: httpClient}
}

func (client *krakenClient) GetName() string {
	return "Kraken"
}

// Check to see if we have error in the response
func (client *krakenClient) extractError(respByte []byte) error {
	errorArray := gjson.GetBytes(respByte, "error").Array()
	if len(errorArray) > 0 {
		errMsg := errorArray[0].Get("0").String()
		if len(errMsg) != 0 {
			return errors.New(errMsg)
		}
	}
	return nil
}

func (client *krakenClient) GetKlinePrice(symbol string, since time.Time, interval int) (float64, error) {
	symbolUpperCase := strings.ToUpper(symbol)
	respByte, err := client.Get(krakenBaseApi+"OHLC", map[string]string{
		"pair":     symbolUpperCase,
		"since":    strconv.FormatInt(since.Unix(), 10),
		"interval": strconv.Itoa(interval),
	})
	if err := client.extractError(respByte); err != nil {
		return 0, fmt.Errorf("kraken get kline: %w", err)
	}
	if err != nil {
		return 0, err
	}

	// gjson saved my life, no need to struggle with different/weird response types
	candleV := gjson.GetBytes(respByte, fmt.Sprintf("result.%s.0", strings.ToUpper(symbol))).Array()
	if len(candleV) != 8 {
		return 0, fmt.Errorf("kraken malformed kline response, expecting 8 elements, got %d", len(candleV))
	}

	timestamp := candleV[0].Int()
	openPrice := candleV[1].Float()
	logrus.Debugf("%s - Kline for %s uses open price at %s", client.GetName(), since.Local(),
		time.Unix(timestamp, 0).Local())
	return openPrice, nil
}

func (client *krakenClient) GetSymbolPrice(symbol string) (*model.SymbolPrice, error) {
	respByte, err := client.Get(krakenBaseApi+"Ticker", map[string]string{"pair": strings.ToUpper(symbol)})
	if err := client.extractError(respByte); err != nil {
		return nil, fmt.Errorf("kraken get ticker: %w", err)
	}
	if err != nil {
		return nil, err
	}

	lastPriceV := gjson.GetBytes(respByte, fmt.Sprintf("result.%s.c.0", strings.ToUpper(symbol)))
	if !lastPriceV.Exists() {
		return nil, fmt.Errorf("kraken malformed ticker response, missing key %s", fmt.Sprintf("result.%s.c.0", strings.ToUpper(symbol)))
	}
	lastPrice := lastPriceV.Float()

	time.Sleep(time.Second) // API call rate limit
	var (
		now              = time.Now()
		percentChange1h  = math.MaxFloat64
		percentChange24h = math.MaxFloat64
	)
	price1hAgo, err := client.GetKlinePrice(symbol, now.Add(-61*time.Minute), 1)
	if err != nil {
		logrus.Warnf("%s - Failed to get price 1 hour ago, error: %v\n", client.GetName(), err)
	} else if price1hAgo != 0 {
		percentChange1h = (lastPrice - price1hAgo) / price1hAgo * 100
	}
	price24hAgo, err := client.GetKlinePrice(symbol, now.Add(-24*time.Hour), 5)
	if err != nil {
		logrus.Warnf("%s - Failed to get price 24 hours ago, error: %v\n", client.GetName(), err)
	} else if price24hAgo != 0 {
		percentChange24h = (lastPrice - price24hAgo) / price24hAgo * 100
	}

	return &model.SymbolPrice{
		Symbol:           symbol,
		Price:            lastPriceV.String(),
		UpdateAt:         time.Now(),
		Source:           client.GetName(),
		PercentChange1h:  percentChange1h,
		PercentChange24h: percentChange24h,
	}, nil
}

func init() {
	Register(NewKrakenClient)
}
