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

// https://github.com/huobiapi/API_Docs/wiki/REST_api_reference
const huobiBaseApi = "https://api.huobipro.com"

type huobiClient struct {
	exchangeBaseClient
	AccessKey string
	SecretKey string
}

type huobiCommonResponse struct {
	Status  string
	Ts      int64
	ErrMsg  string `json:"err-msg"`
	ErrCode string `json:"err-code"`
}

type huobiTickerResponse struct {
	huobiCommonResponse
	Tick struct {
		//Id   string
		Ts   int64
		Data []struct {
			//Id    string
			Price float64
			Ts    int64
		}
	}
}

func (resp *huobiTickerResponse) getCommonResponse() huobiCommonResponse {
	return resp.huobiCommonResponse
}

type huobiKlineResponse struct {
	huobiCommonResponse
	Data []struct {
		Open float64
	}
}

func (resp *huobiKlineResponse) getCommonResponse() huobiCommonResponse {
	return resp.huobiCommonResponse
}

// Any way to hold the common response, instead of adding an interface here?
type huobiCommonResponseProvider interface {
	getCommonResponse() huobiCommonResponse
}

func NewHuobiClient(httpClient *http.Client) ExchangeClient {
	return &huobiClient{exchangeBaseClient: *newExchangeBase(huobiBaseApi, httpClient)}
}

func (client *huobiClient) GetName() string {
	return "Huobi"
}

func (client *huobiClient) decodeResponse(body io.ReadCloser, respJSON huobiCommonResponseProvider) error {
	defer body.Close()

	decoder := json.NewDecoder(body)
	if err := decoder.Decode(&respJSON); err != nil {
		return err
	}

	// All I need is to get the common part, I don't like this
	commonResponse := respJSON.getCommonResponse()
	if strings.ToLower(commonResponse.Status) != "ok" {
		if commonResponse.ErrMsg == "" {
			commonResponse.ErrMsg = "unknown error message"
		}
		return errors.New(commonResponse.ErrMsg)
	}
	return nil
}

func (client *huobiClient) GetKlinePrice(symbol, period string, size int) (float64, error) {
	symbol = strings.ToLower(symbol)
	rawUrl := client.buildUrl("/market/history/kline", map[string]string{
		"symbol": symbol,
		"period": period,
		"size":   strconv.Itoa(size),
	})
	resp, err := client.HTTPClient.Get(rawUrl)
	if err != nil {
		return 0, err
	}

	var respJSON huobiKlineResponse
	err = client.decodeResponse(resp.Body, &respJSON)
	if err != nil {
		return 0, err
	}
	return respJSON.Data[len(respJSON.Data)-1].Open, nil
}

func (client *huobiClient) GetSymbolPrice(symbol string) (*SymbolPrice, error) {
	rawUrl := client.buildUrl("/market/trade", map[string]string{"symbol": strings.ToLower(symbol)})
	resp, err := client.HTTPClient.Get(rawUrl)
	if err != nil {
		return nil, err
	}

	var respJSON huobiTickerResponse
	err = client.decodeResponse(resp.Body, &respJSON)
	if err != nil {
		return nil, err
	}

	ticker := respJSON.Tick.Data[len(respJSON.Tick.Data)-1] // Use the last one

	var percentChange1h, percentChange24h float64
	price1hAgo, err := client.GetKlinePrice(symbol, "1min", 60)
	if err != nil {
		logrus.Warnf("%s - Failed to get price 1 hour ago, error: %v\n", client.GetName(), err)
	} else if price1hAgo != 0 {
		percentChange1h = (ticker.Price - price1hAgo) / price1hAgo * 100
	}

	price24hAgo, err := client.GetKlinePrice(symbol, "60min", 24)
	if err != nil {
		logrus.Warnf("%s - Failed to get price 24 hours ago, error: %v\n", client.GetName(), err)
	} else if price24hAgo != 0 {
		percentChange24h = (ticker.Price - price24hAgo) / price24hAgo * 100
	}

	return &SymbolPrice{
		Symbol:           symbol,
		Price:            strconv.FormatFloat(ticker.Price, 'f', -1, 64),
		UpdateAt:         time.Unix(ticker.Ts/1000, 0),
		Source:           client.GetName(),
		PercentChange1h:  percentChange1h,
		PercentChange24h: percentChange24h,
	}, nil
}

func init() {
	register((&huobiClient{}).GetName(), NewHuobiClient)
}
