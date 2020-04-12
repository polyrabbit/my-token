package exchange

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/polyrabbit/my-token/exchange/model"
	"github.com/polyrabbit/my-token/http"
	"github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

// https://www.okex.com/docs/zh/#spot-some
const okexBaseApi = "https://www.okex.com/api/spot/v3/instruments/"

type okexClient struct {
	AccessKey string
	SecretKey string
}

func (client *okexClient) GetName() string {
	return "OKEx"
}

func (client *okexClient) GetKlinePrice(symbol, granularity string, start, end time.Time) (float64, error) {
	respByte, err := http.Get(okexBaseApi+symbol+"/candles", map[string]string{
		"granularity": granularity,
		"start":       start.UTC().Format(time.RFC3339),
		"end":         end.UTC().Format(time.RFC3339),
	})
	if err := client.extractError(respByte); err != nil {
		return 0, fmt.Errorf("okex get candles: %w", err)
	}
	if err != nil {
		return 0, fmt.Errorf("okex get candles: %w", err)
	}

	klines := gjson.ParseBytes(respByte).Array()
	if len(klines) == 0 {
		return 0, fmt.Errorf("okex got empty candles response")
	}
	lastKline := klines[len(klines)-1]
	if len(lastKline.Array()) != 6 {
		return 0, fmt.Errorf(`okex malformed kline response, got size %d`, len(lastKline.Array()))
	}
	updated := time.Now()
	if parsed, err := time.Parse(time.RFC3339, lastKline.Get("0").String()); err == nil {
		updated = parsed
	}
	logrus.Debugf("%s - Kline for %s seconds uses price at %s",
		client.GetName(), granularity, updated.Local())
	return lastKline.Get("1").Float(), nil
}

func (client *okexClient) GetSymbolPrice(symbol string) (*model.SymbolPrice, error) {
	respByte, err := http.Get(okexBaseApi+symbol+"/ticker", nil)
	if err := client.extractError(respByte); err != nil {
		// Extract more readable first if have
		return nil, fmt.Errorf("okex get symbol price: %w", err)
	}
	if err != nil {
		return nil, fmt.Errorf("okex get symbol price: %w", err)
	}
	lastV := gjson.GetBytes(respByte, "last")
	if !lastV.Exists() {
		return nil, fmt.Errorf(`okex malformed get symbol price response, missing "last" key`)
	}
	lastPrice := lastV.Float()
	updateAtV := gjson.GetBytes(respByte, "timestamp")
	if !updateAtV.Exists() {
		return nil, fmt.Errorf(`okex malformed get symbol price response, missing "timestamp" key`)
	}
	updateAt, err := time.Parse(time.RFC3339, updateAtV.String())
	if err != nil {
		return nil, fmt.Errorf("okex parse timestamp: %w", err)
	}

	var percentChange1h, percentChange24h = math.MaxFloat64, math.MaxFloat64
	price1hAgo, err := client.GetKlinePrice(symbol, "60", updateAt.Add(-time.Hour), updateAt)
	if err != nil {
		logrus.Warnf("%s - Failed to get price 1 hour ago, error: %v\n", client.GetName(), err)
	} else if price1hAgo != 0 {
		percentChange1h = (lastPrice - price1hAgo) / price1hAgo * 100
	}

	price24hAgo, err := client.GetKlinePrice(symbol, "900", updateAt.Add(-24*time.Hour), updateAt)
	if err != nil {
		logrus.Warnf("%s - Failed to get price 24 hours ago, error: %v\n", client.GetName(), err)
	} else if price24hAgo != 0 {
		percentChange24h = (lastPrice - price24hAgo) / price24hAgo * 100
	}

	return &model.SymbolPrice{
		Symbol:           symbol,
		Price:            strconv.FormatFloat(lastPrice, 'f', -1, 64),
		UpdateAt:         updateAt,
		Source:           client.GetName(),
		PercentChange1h:  percentChange1h,
		PercentChange24h: percentChange24h,
	}, nil
}

// Check to see if we have error in the response
func (client *okexClient) extractError(respByte []byte) error {
	errorMsg := gjson.GetBytes(respByte, "error_message")
	if !errorMsg.Exists() {
		errorMsg = gjson.GetBytes(respByte, "message")
	}
	if len(errorMsg.String()) != 0 {
		return errors.New(errorMsg.String())
	}
	return nil
}

func init() {
	model.Register(new(okexClient))
}
