package exchange

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// https://developer.big.one/
const bigOneBaseApi = "https://api.big.one/"

type bigOneClient struct {
	exchangeBaseClient
	AccessKey string
	SecretKey string
}

type bigOneErrorResponse struct {
	Error *struct {
		Status      int
		Code        int
		Description string
	}
}

type bigOneMarketResponse struct {
	bigOneErrorResponse
	Data struct {
		Symbol string
		Ticker struct {
			Price float64 `json:",string"`
		}
		//Metrics map[string][]interface{}
		Metrics struct {
			// timestamp, open, close, high, low, volume
			Min1  [][]interface{} `json:"0000001"`
			Min5  [][]interface{} `json:"0000005"`
			Min15 [][]interface{} `json:"0000015"`
		}
	}
}

func NewBigOneClient(httpClient *http.Client) *bigOneClient {
	return &bigOneClient{exchangeBaseClient: *newExchangeBase(bigOneBaseApi, httpClient)}
}

func (client *bigOneClient) GetName() string {
	return "BigONE"
}

func (client *bigOneClient) decodeResponse(body io.ReadCloser, respJSON zbCommonResponseProvider) error {
	defer body.Close()

	decoder := json.NewDecoder(body)
	if err := decoder.Decode(&respJSON); err != nil {
		return err
	}

	// All I need is to get the common part, I don't like this
	commonResponse := respJSON.getCommonResponse()
	if commonResponse.Error != nil {
		return errors.New(*commonResponse.Error)
	}
	if commonResponse.Message != nil {
		return errors.New(*commonResponse.Message)
	}
	return nil
}

func (client *bigOneClient) SearchKlinePriceNear(klineIntervals [][]interface{}, after time.Time) (float64, error) {
	var intervalTime time.Time
	for _, interval := range klineIntervals {
		if ts, ok := interval[0].(float64); ok {
			intervalTime = time.Unix(int64(ts)/1000, 0)
			if after.Equal(intervalTime) || after.After(intervalTime) {
				// Assume candles are sorted in asc order, so the first less than or equal to is the candle looking for
				logrus.Debugf("%s - Kline for %v uses open price at %v", client.GetName(), after.Local(), intervalTime.Local())
				if openStr, ok := interval[1].(string); ok {
					return strconv.ParseFloat(openStr, 64)
				} else {
					return 0, fmt.Errorf("cannot convert open price item %v of kline to string", interval[1])
				}
			}
		} else {
			return 0, fmt.Errorf("cannot convert first item %v of kline to float64", interval[0])
		}
	}
	return 0, fmt.Errorf("no time found right after %v, the last time in this interval is %v", after.Local(), intervalTime.Local())
}

func (client *bigOneClient) GetSymbolPrice(symbol string) (*SymbolPrice, error) {
	// One api to get all
	resp, err := client.httpGet("markets/"+strings.ToUpper(symbol), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var respJSON bigOneMarketResponse

	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&respJSON); err != nil {
		return nil, err
	}

	if respJSON.Error != nil {
		return nil, errors.New(respJSON.Error.Description)
	}

	var (
		now                               = time.Now()
		percentChange1h, percentChange24h = math.MaxFloat64, math.MaxFloat64
	)
	price1hAgo, err := client.SearchKlinePriceNear(respJSON.Data.Metrics.Min1, now.Add(-1*time.Hour))
	if price1hAgo == 0 {
		// BigOne has a very low volume, that prices after certain amount are all zero, so enlarge intervals here.
		price1hAgo, err = client.SearchKlinePriceNear(respJSON.Data.Metrics.Min5, now.Add(-1*time.Hour))
	}
	if err != nil {
		logrus.Warnf("%s - Failed to get price 1 hour ago, error: %v\n", client.GetName(), err)
	} else if price1hAgo != 0 {
		percentChange1h = (respJSON.Data.Ticker.Price - price1hAgo) / price1hAgo * 100
	}

	price24hAgo, err := client.SearchKlinePriceNear(respJSON.Data.Metrics.Min15, now.Add(-24*time.Hour))
	if err != nil {
		logrus.Warnf("%s - Failed to get price 24 hours ago, error: %v\n", client.GetName(), err)
	} else if price24hAgo != 0 {
		percentChange24h = (respJSON.Data.Ticker.Price - price24hAgo) / price24hAgo * 100
	}

	return &SymbolPrice{
		Symbol:           symbol,
		Price:            strconv.FormatFloat(respJSON.Data.Ticker.Price, 'f', -1, 64),
		UpdateAt:         time.Now(),
		Source:           client.GetName(),
		PercentChange1h:  percentChange1h,
		PercentChange24h: percentChange24h,
	}, nil
}

func init() {
	register((&bigOneClient{}).GetName(), func(client *http.Client) ExchangeClient {
		// Limited by type system in Go, I hate wrapper/adapter
		return NewBigOneClient(client)
	})
}
