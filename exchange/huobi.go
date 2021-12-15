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

// https://github.com/huobiapi/API_Docs/wiki/REST_api_reference
const huobiBaseApi = "https://api.huobipro.com"

type huobiClient struct {
    *http.Client
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
        Open interface{}
    }
}

func (resp *huobiKlineResponse) getCommonResponse() huobiCommonResponse {
    return resp.huobiCommonResponse
}

// Any way to hold the common response, instead of adding an interface here?
type huobiCommonResponseProvider interface {
    getCommonResponse() huobiCommonResponse
}

func NewHuobiClient(queries map[string]*config.PriceQuery, httpClient *http.Client) ExchangeClient {
    return &huobiClient{Client: httpClient}
}

func (client *huobiClient) GetName() string {
    return "Huobi"
}

func (client *huobiClient) decodeResponse(respBytes []byte, respJSON huobiCommonResponseProvider) error {
    if err := json.Unmarshal(respBytes, &respJSON); err != nil {
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
    respByte, err := client.Get(huobiBaseApi+"/market/history/kline", http.WithQuery(map[string]string{
        "symbol": symbol,
        "period": period,
        "size":   strconv.Itoa(size),
    }))
    if err != nil {
        return 0, err
    }

    var respJSON huobiKlineResponse
    err = client.decodeResponse(respByte, &respJSON)
    if err != nil {
        return 0, err
    }
    var open float64
    if r, ok := respJSON.Data[len(respJSON.Data)-1].Open.(float64); ok {
        open = r
    }
    if r, ok := respJSON.Data[len(respJSON.Data)-1].Open.(string); ok {
        open, _ = strconv.ParseFloat(r, 64)
    }
    return open, nil
}

func (client *huobiClient) GetSymbolPrice(symbol string) (*SymbolPrice, error) {
    respByte, err := client.Get(huobiBaseApi+"/market/trade", http.WithQuery(map[string]string{"symbol": strings.ToLower(symbol)}))
    if err != nil {
        return nil, err
    }

    var respJSON huobiTickerResponse
    err = client.decodeResponse(respByte, &respJSON)
    if err != nil {
        return nil, err
    }

    ticker := respJSON.Tick.Data[len(respJSON.Tick.Data)-1] // Use the last one

    var percentChange1h, percentChange24h = math.MaxFloat64, math.MaxFloat64
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
    Register(NewHuobiClient)
}
