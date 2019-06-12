package exchange

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// https://support.bittrex.com/hc/en-us/articles/115003723911
const bittrexBaseApi = "https://bittrex.com/api/v1.1/"

// https://github.com/thebotguys/golang-bittrex-api/wiki/Bittrex-API-Reference-(Unofficial)
const bittrexV2BaseApi = "https://bittrex.com/Api/v2.0/pub/market/"

type bittrexClient struct {
	exchangeBaseClient
	AccessKey string
	SecretKey string
}

type bittrexCommonResponse struct {
	Success bool
	Message string
}

type bittrexTickerResponse struct {
	bittrexCommonResponse
	Result struct {
		Last float64
	}
}

type bittrexKlineResponse struct {
	bittrexCommonResponse
	Result []struct {
		High       float64 `json:"H"`
		Open       float64 `json:"O"`
		Close      float64 `json:"C"`
		Low        float64 `json:"L"`
		Volume     float64 `json:"V"`
		BaseVolume float64 `json:"BV"`
		Timestamp  string  `json:"T"`
	}
}

func (resp *bittrexTickerResponse) getCommonResponse() bittrexCommonResponse {
	return resp.bittrexCommonResponse
}

func (resp *bittrexKlineResponse) getCommonResponse() bittrexCommonResponse {
	return resp.bittrexCommonResponse
}

// Any way to hold the common response, instead of adding an interface here?
type bittrexCommonResponseProvider interface {
	getCommonResponse() bittrexCommonResponse
}

func NewBittrexClient(httpClient *http.Client) *bittrexClient {
	return &bittrexClient{exchangeBaseClient: *newExchangeBase(bittrexBaseApi, httpClient)}
}

func (client *bittrexClient) GetName() string {
	return "Bittrex"
}

func (client *bittrexClient) decodeResponse(resp *http.Response, respJSON bittrexCommonResponseProvider) error {
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(respJSON); err != nil {
		if resp.StatusCode != 200 {
			return errors.New(resp.Status)
		}
		return err
	}

	// All I need is to get the common part, I don't like this
	commonResponse := respJSON.getCommonResponse()
	if !commonResponse.Success {
		return errors.New(commonResponse.Message)
	}
	return nil
}

func (client *bittrexClient) GetKlineTicks(market, interval string) (*bittrexKlineResponse, error) {
	// Using a unofficial api
	origUrl := client.BaseURL
	defer func() { client.BaseURL = origUrl }() // Restore to v1 (stable)
	client.BaseURL, _ = url.Parse(bittrexV2BaseApi)

	market = strings.ToLower(market)
	resp, err := client.httpGet("GetTicks", map[string]string{
		"marketName":   market,
		"tickInterval": interval,
	})
	if err != nil {
		return nil, err
	}

	var respJSON bittrexKlineResponse
	err = client.decodeResponse(resp, &respJSON)
	if err != nil {
		return nil, err
	}
	return &respJSON, nil
}

func (client *bittrexClient) GetPriceRightAfter(klineResp *bittrexKlineResponse, after time.Time) (float64, error) {
	for _, candle := range klineResp.Result {
		candleTime, err := time.Parse("2006-01-02T15:04:05", candle.Timestamp)
		if err == nil {
			if after.Equal(candleTime) || after.Before(candleTime) {
				// Assume candles are sorted in asc order, so the first less than or equal to is the candle looking for
				logrus.Debugf("%s - Kline for %v uses open price at %v", client.GetName(), after.Local(), candleTime.Local())
				return candle.Open, nil
			}
		}
	}
	return 0, fmt.Errorf("no time found right after %v", after)
}

func (client *bittrexClient) GetSymbolPrice(symbol string) (*SymbolPrice, error) {
	resp, err := client.httpGet("public/getticker", map[string]string{"market": strings.ToUpper(symbol)})
	if err != nil {
		return nil, err
	}

	var respJSON bittrexTickerResponse
	err = client.decodeResponse(resp, &respJSON)
	if err != nil {
		return nil, err
	}

	var percentChange1h, percentChange24h = math.MaxFloat64, math.MaxFloat64
	klineResp, err := client.GetKlineTicks(symbol, "thirtyMin") // oneMin, fiveMin are too large, bittrex doesn't support filter
	if err != nil {
		logrus.Warnf("%s - Failed to get kline ticks, error: %v", client.GetName(), err)
	} else {
		now := time.Now()
		sort.Slice(klineResp.Result, func(i, j int) bool {
			return klineResp.Result[i].Timestamp < klineResp.Result[j].Timestamp
		})

		lastHour := now.Add(-1 * time.Hour)
		price1hAgo, err := client.GetPriceRightAfter(klineResp, lastHour)
		if err != nil {
			logrus.Warnf("%s - Failed to get price 1 hour ago, error: %v\n", client.GetName(), err)
		} else if price1hAgo != 0 {
			percentChange1h = (respJSON.Result.Last - price1hAgo) / price1hAgo * 100
		}

		last24Hour := now.Add(-24 * time.Hour)
		price24hAgo, err := client.GetPriceRightAfter(klineResp, last24Hour)
		if err != nil {
			logrus.Warnf("%s - Failed to get price 24 hours ago, error: %v\n", client.GetName(), err)
		} else if price24hAgo != 0 {
			percentChange24h = (respJSON.Result.Last - price24hAgo) / price24hAgo * 100
		}
	}

	return &SymbolPrice{
		Symbol:           symbol,
		Price:            strconv.FormatFloat(respJSON.Result.Last, 'f', -1, 64),
		UpdateAt:         time.Now(),
		Source:           client.GetName(),
		PercentChange1h:  percentChange1h,
		PercentChange24h: percentChange24h,
	}, nil
}

func init() {
	register((&bittrexClient{}).GetName(), func(client *http.Client) ExchangeClient {
		// Limited by type system in Go, I hate wrapper/adapter
		return NewBittrexClient(client)
	})
}
