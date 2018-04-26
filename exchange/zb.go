package exchange

import (
	"encoding/json"
	"errors"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// https://www.zb.com/i/developer
const zbBaseApi = "http://api.zb.com/data/v1/"

type zbClient struct {
	exchangeBaseClient
	AccessKey string
	SecretKey string
}

type zbCommonResponse struct {
	Error   *string
	Message *string
}

type zbTickerResponse struct {
	zbCommonResponse
	Date   int64 `json:",string"`
	Ticker struct {
		Last float64 `json:",string"`
	}
}

type zbKlineResponse struct {
	zbCommonResponse
	Data [][]float64
}

func (resp *zbTickerResponse) getCommonResponse() zbCommonResponse {
	return resp.zbCommonResponse
}

func (resp *zbKlineResponse) getCommonResponse() zbCommonResponse {
	return resp.zbCommonResponse
}

// Any way to hold the common response, instead of adding an interface here?
type zbCommonResponseProvider interface {
	getCommonResponse() zbCommonResponse
}

func NewZBClient(httpClient *http.Client) *zbClient {
	return &zbClient{exchangeBaseClient: *newExchangeBase(zbBaseApi, httpClient)}
}

func (client *zbClient) GetName() string {
	return "ZB"
}

func (client *zbClient) decodeResponse(body io.ReadCloser, respJSON zbCommonResponseProvider) error {
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

func (client *zbClient) GetKlinePrice(symbol, period string, size int) (float64, error) {
	symbol = strings.ToLower(symbol)
	rawUrl := client.buildUrl("kline", map[string]string{
		"market": symbol,
		"type":   period,
		"size":   strconv.Itoa(size),
	})
	resp, err := client.HTTPClient.Get(rawUrl)
	if err != nil {
		return 0, err
	}

	var respJSON zbKlineResponse
	err = client.decodeResponse(resp.Body, &respJSON)
	if err != nil {
		return 0, err
	}
	logrus.Debugf("%s - Kline for %s*%v uses price at %s", client.GetName(), period, size,
		time.Unix(int64(respJSON.Data[0][0])/1000, 0))
	return respJSON.Data[0][1], nil
}

func (client *zbClient) GetSymbolPrice(symbol string) (*SymbolPrice, error) {
	rawUrl := client.buildUrl("ticker", map[string]string{"market": strings.ToLower(symbol)})
	resp, err := client.HTTPClient.Get(rawUrl)
	if err != nil {
		return nil, err
	}

	var respJSON zbTickerResponse
	err = client.decodeResponse(resp.Body, &respJSON)
	if err != nil {
		return nil, err
	}

	var percentChange1h, percentChange24h float64
	price1hAgo, err := client.GetKlinePrice(symbol, "1min", 60)
	if err != nil {
		logrus.Warnf("%s - Failed to get price 1 hour ago, error: %v\n", client.GetName(), err)
	} else if price1hAgo != 0 {
		percentChange1h = (respJSON.Ticker.Last - price1hAgo) / price1hAgo * 100
	}

	time.Sleep(time.Second)                                       // ZB limits 1 req/sec for Kline
	price24hAgo, err := client.GetKlinePrice(symbol, "3min", 489) // Why not 480?
	if err != nil {
		logrus.Warnf("%s - Failed to get price 24 hours ago, error: %v\n", client.GetName(), err)
	} else if price24hAgo != 0 {
		percentChange24h = (respJSON.Ticker.Last - price24hAgo) / price24hAgo * 100
	}

	return &SymbolPrice{
		Symbol:           symbol,
		Price:            strconv.FormatFloat(respJSON.Ticker.Last, 'f', -1, 64),
		UpdateAt:         time.Unix(respJSON.Date/1000, 0),
		Source:           client.GetName(),
		PercentChange1h:  percentChange1h,
		PercentChange24h: percentChange24h,
	}, nil
}
