package exchange

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// https://docs.bitfinex.com/v2/docs
const bitfinixBaseApi = "https://api.bitfinex.com/v2/" //Need api v2 to get kline

type bitfinixClient struct {
	exchangeBaseClient
	AccessKey string
	SecretKey string
}

func NewBitfinixClient(httpClient *http.Client) *bitfinixClient {
	return &bitfinixClient{exchangeBaseClient: *newExchangeBase(bitfinixBaseApi, httpClient)}
}

func (client *bitfinixClient) GetName() string {
	return "Bitfinex"
}

func (client *bitfinixClient) readResponse(resp *http.Response) ([]byte, error) {
	respBytes, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return nil, err
	}

	var errResp []interface{}

	if err := json.Unmarshal(respBytes, &errResp); err != nil {
		if resp.StatusCode != 200 {
			return nil, errors.New(resp.Status)
		}
		return nil, err
	}

	if len(errResp) == 0 {
		return nil, fmt.Errorf("empty response")
	}

	if errstr, ok := errResp[0].(string); ok && errstr == "error" {
		return nil, fmt.Errorf(errResp[2].(string))
	}

	return respBytes, nil
}

func (client *bitfinixClient) GetKlinePrice(symbol, frame string, start time.Time) (float64, error) {
	candlePath := fmt.Sprintf("candles/trade:%s:t%s/hist", frame, symbol)
	rawUrl := client.buildUrl(candlePath, map[string]string{
		"start": strconv.FormatInt(start.Unix()*1000, 10),
		"sort":  "1",
		"limit": "1",
	})
	resp, err := client.HTTPClient.Get(rawUrl)
	if err != nil {
		return 0, err
	}
	respBytes, err := client.readResponse(resp)

	var klineResp [][]float64

	if err := json.Unmarshal(respBytes, &klineResp); err != nil {
		return 0, err
	}
	logrus.Debugf("%s - %s Kline for %s uses price at %s", client.GetName(), symbol, start,
		time.Unix(int64(klineResp[0][0])/1000, 0))
	return klineResp[0][1], nil
}

func (client *bitfinixClient) GetSymbolPrice(symbol string) (*SymbolPrice, error) {
	symbol = strings.ToUpper(symbol)
	rawUrl := client.buildUrl("ticker/t"+symbol, map[string]string{})
	resp, err := client.HTTPClient.Get(rawUrl)
	if err != nil {
		return nil, err
	}
	respBytes, err := client.readResponse(resp)

	var tickerResp []float64

	if err := json.Unmarshal(respBytes, &tickerResp); err != nil {
		return nil, err
	}

	if len(tickerResp) < 7 {
		return nil, fmt.Errorf("[%s] - not enough data in response array, get %v", client.GetName(), tickerResp)
	}

	currentPrice := tickerResp[6]

	var percentChange1h, percentChange24h = math.MaxFloat64, math.MaxFloat64

	now := time.Now()

	lastHour := now.Add(-1 * time.Hour)
	lastHourPrice, err := client.GetKlinePrice(symbol, "1m", lastHour)
	if err != nil {
		logrus.Warnf("Failed to get price 1 hour ago, error: %s\n", err)
	}
	if lastHourPrice != 0 {
		percentChange1h = (currentPrice - lastHourPrice) / lastHourPrice * 100
	}

	lastDay := now.Add(-24 * time.Hour)
	lastDayPrice, err := client.GetKlinePrice(symbol, "1m", lastDay)
	if err != nil {
		logrus.Warnf("Failed to get price 24 hour ago, error: %s\n", err)
	}
	if lastDayPrice != 0 {
		percentChange24h = (currentPrice - lastDayPrice) / lastDayPrice * 100
	}

	return &SymbolPrice{
		Symbol:           strings.ToUpper(symbol),
		Price:            strconv.FormatFloat(currentPrice, 'f', -1, 64),
		Source:           client.GetName(),
		UpdateAt:         time.Now(),
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
