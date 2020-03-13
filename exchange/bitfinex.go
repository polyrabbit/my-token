package exchange

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/polyrabbit/my-token/exchange/model"

	"github.com/polyrabbit/my-token/http"
	"github.com/sirupsen/logrus"
)

// https://docs.bitfinex.com/v2/docs
const bitfinixBaseApi = "https://api.bitfinex.com/v2/" //Need api v2 to get kline

type bitfinixClient struct {
	AccessKey string
	SecretKey string
}

func (client *bitfinixClient) GetName() string {
	return "Bitfinex"
}

func (client *bitfinixClient) checkError(respContent []byte) error {
	var errResp []interface{}
	if err := json.Unmarshal(respContent, &errResp); err != nil {
		return err
	}

	if len(errResp) == 0 {
		return errors.New("empty response")
	}

	if errstr, ok := errResp[0].(string); ok && errstr == "error" {
		return errors.New(errResp[2].(string))
	}

	return nil
}

func (client *bitfinixClient) GetKlinePrice(symbol, frame string, start time.Time) (float64, error) {
	candlePath := fmt.Sprintf("candles/trade:%s:t%s/hist", frame, symbol)
	respBytes, err := http.Get(bitfinixBaseApi+candlePath, map[string]string{
		"start": strconv.FormatInt(start.Unix()*1000, 10),
		"sort":  "1",
		"limit": "1",
	})
	if err != nil {
		return 0, err
	}
	if err := client.checkError(respBytes); err != nil {
		return 0, err
	}

	var klineResp [][]float64
	if err := json.Unmarshal(respBytes, &klineResp); err != nil {
		return 0, err
	}
	logrus.Debugf("%s - %s Kline for %s uses price at %s", client.GetName(), symbol, start,
		time.Unix(int64(klineResp[0][0])/1000, 0))
	return klineResp[0][1], nil
}

func (client *bitfinixClient) GetSymbolPrice(symbol string) (*model.SymbolPrice, error) {
	symbol = strings.ToUpper(symbol)
	respBytes, err := http.Get(bitfinixBaseApi+"ticker/t"+symbol, nil)
	if err != nil {
		return nil, err
	}
	if err := client.checkError(respBytes); err != nil {
		return nil, err
	}

	var tickerResp []float64
	if err := json.Unmarshal(respBytes, &tickerResp); err != nil {
		return nil, err
	}
	if len(tickerResp) < 7 {
		return nil, fmt.Errorf("[%s] - not enough data in response array, get %v", client.GetName(), tickerResp)
	}

	percentChange1h, percentChange24h := math.MaxFloat64, math.MaxFloat64
	currentPrice := tickerResp[6]
	now := time.Now()
	lastHour := now.Add(-1 * time.Hour)
	lastHourPrice, err := client.GetKlinePrice(symbol, "1m", lastHour)
	if err != nil {
		logrus.Warnf("Failed to get price 1 hour ago, error: %v", err)
	}
	if lastHourPrice != 0 {
		percentChange1h = (currentPrice - lastHourPrice) / lastHourPrice * 100
	}

	lastDay := now.Add(-24 * time.Hour)
	lastDayPrice, err := client.GetKlinePrice(symbol, "1m", lastDay)
	if err != nil {
		logrus.Warnf("Failed to get price 24 hour ago, error: %v", err)
	}
	if lastDayPrice != 0 {
		percentChange24h = (currentPrice - lastDayPrice) / lastDayPrice * 100
	}

	return &model.SymbolPrice{
		Symbol:           strings.ToUpper(symbol),
		Price:            strconv.FormatFloat(currentPrice, 'f', -1, 64),
		Source:           client.GetName(),
		UpdateAt:         time.Now(),
		PercentChange1h:  percentChange1h,
		PercentChange24h: percentChange24h,
	}, nil
}

func init() {
	model.Register(&bitfinixClient{})
}
