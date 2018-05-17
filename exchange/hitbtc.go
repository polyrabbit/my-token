package exchange

import (
	"encoding/json"
	"errors"
	"github.com/sirupsen/logrus"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// https://api.hitbtc.com/#market-data
const hitBtcBaseApi = "https://api.hitbtc.com/api/2/"

// ZB api is very similar to OKEx, who copied whom?

type hitBtcClient struct {
	exchangeBaseClient
	AccessKey string
	SecretKey string
}

type hitBtcCommonResponse struct {
	Error *struct {
		Code        int
		Message     string
		Description string
	}
}

type hitBtcTickerResponse struct {
	hitBtcCommonResponse
	Last      float64 `json:",string"`
	Open      float64 `json:",string"`
	Timestamp string
}

type hitBtcKlineResponse struct {
	hitBtcCommonResponse
	Timestamp string
	Open      float64 `json:",string"`
}

func (resp *hitBtcTickerResponse) getCommonResponse() hitBtcCommonResponse {
	return resp.hitBtcCommonResponse
}

func (resp *hitBtcKlineResponse) getCommonResponse() hitBtcCommonResponse {
	return resp.hitBtcCommonResponse
}

// Any way to hold the common response, instead of adding an interface here?
type hitBtcCommonResponseProvider interface {
	getCommonResponse() hitBtcCommonResponse
}

func NewHitBtcClient(httpClient *http.Client) *hitBtcClient {
	return &hitBtcClient{exchangeBaseClient: *newExchangeBase(hitBtcBaseApi, httpClient)}
}

func (client *hitBtcClient) GetName() string {
	return "HitBTC"
}

func (client *hitBtcClient) decodeResponse(resp *http.Response, respJSON hitBtcCommonResponseProvider) error {
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&respJSON); err != nil {
		if resp.StatusCode != 200 {
			return errors.New(resp.Status)
		}
		return err
	}

	// All I need is to get the common part, I don't like this
	commonResponse := respJSON.getCommonResponse()
	if commonResponse.Error != nil {
		return errors.New(commonResponse.Error.Message + " - " + commonResponse.Error.Description)
	}
	return nil
}

func (client *hitBtcClient) GetKlinePrice(symbol, period string, limit int) (float64, error) {
	symbol = strings.ToLower(symbol)
	resp, err := client.httpGet("public/candles/"+strings.ToUpper(symbol), map[string]string{
		"period": period,
		"limit":  strconv.Itoa(limit),
	})
	if err != nil {
		return 0, err
	}

	if resp.StatusCode != 200 {
		var respJSON hitBtcKlineResponse
		return 0, client.decodeResponse(resp, &respJSON)
	}
	var respJSON []hitBtcKlineResponse
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&respJSON); err != nil {
		if resp.StatusCode != 200 {
			return 0, errors.New(resp.Status)
		}
		return 0, err
	}
	logrus.Debugf("%s - Kline for %s*%v uses price at %s", client.GetName(), period, limit, respJSON[0].Timestamp)
	return respJSON[0].Open, nil
}

func (client *hitBtcClient) GetSymbolPrice(symbol string) (*SymbolPrice, error) {
	resp, err := client.httpGet("/public/ticker/"+strings.ToUpper(symbol), nil)
	if err != nil {
		return nil, err
	}

	var respJSON hitBtcTickerResponse
	err = client.decodeResponse(resp, &respJSON)
	if err != nil {
		return nil, err
	}
	updated, err := time.Parse(time.RFC3339, respJSON.Timestamp)
	if err != nil {
		return nil, err
	}

	var percentChange1h, percentChange24h = math.MaxFloat64, math.MaxFloat64
	price1hAgo, err := client.GetKlinePrice(symbol, "M1", 60)
	if err != nil {
		logrus.Warnf("%s - Failed to get price 1 hour ago, error: %v\n", client.GetName(), err)
	} else if price1hAgo != 0 {
		percentChange1h = (respJSON.Last - price1hAgo) / price1hAgo * 100
	}

	//price24hAgo_, err := client.GetKlinePrice(symbol, "M3", 480)
	//logrus.Warnf("%s - %s", price24hAgo_, respJSON.Open)

	price24hAgo := respJSON.Open
	percentChange24h = (respJSON.Last - price24hAgo) / price24hAgo * 100

	return &SymbolPrice{
		Symbol:           symbol,
		Price:            strconv.FormatFloat(respJSON.Last, 'f', -1, 64),
		UpdateAt:         updated,
		Source:           client.GetName(),
		PercentChange1h:  percentChange1h,
		PercentChange24h: percentChange24h,
	}, nil
}

func init() {
	register((&hitBtcClient{}).GetName(), func(client *http.Client) ExchangeClient {
		// Limited by type system in Go, I hate wrapper/adapter
		return NewHitBtcClient(client)
	})
}
