package exchange

import (
	"encoding/json"
	"errors"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/polyrabbit/my-token/config"
	"github.com/polyrabbit/my-token/http"
	"github.com/sirupsen/logrus"
)

// https://api.hitbtc.com/#market-data
const hitBtcBaseApi = "https://api.hitbtc.com/api/2/"

// ZB api is very similar to OKEx, who copied whom?

type hitBtcClient struct {
	*http.Client
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

func NewHitBtcClient(queries map[string]config.PriceQuery, httpClient *http.Client) ExchangeClient {
	return &hitBtcClient{Client: httpClient}
}

func (client *hitBtcClient) GetName() string {
	return "HitBTC"
}

func (client *hitBtcClient) decodeResponse(respBytes []byte, respJSON hitBtcCommonResponseProvider) error {
	if err := json.Unmarshal(respBytes, respJSON); err != nil {
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
	respBytes, err := client.Get(hitBtcBaseApi+"public/candles/"+strings.ToUpper(symbol), http.WithQuery(map[string]string{
		"period": period,
		"limit":  strconv.Itoa(limit),
	}))
	if err != nil {
		return 0, err
	}

	var respJSON []hitBtcKlineResponse
	if err := json.Unmarshal(respBytes, &respJSON); err != nil {
		return 0, err
	}
	logrus.Debugf("%s - Kline for %s*%v uses price at %s", client.GetName(), period, limit, respJSON[0].Timestamp)
	return respJSON[0].Open, nil
}

func (client *hitBtcClient) GetSymbolPrice(symbol string) (*SymbolPrice, error) {
	respBytes, err := client.Get(hitBtcBaseApi + "public/ticker/" + strings.ToUpper(symbol))
	if err != nil {
		return nil, err
	}

	var respJSON hitBtcTickerResponse
	err = client.decodeResponse(respBytes, &respJSON)
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
	Register(NewHitBtcClient)
}
