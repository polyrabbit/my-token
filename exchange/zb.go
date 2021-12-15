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

// https://www.zb.com/i/developer
const zbBaseApi = "http://api.zb.com/data/v1/"

// ZB api is very similar to OKEx, who copied whom?

type zbClient struct {
    *http.Client
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

func NewZBClient(queries map[string]*config.PriceQuery, httpClient *http.Client) ExchangeClient {
    return &zbClient{Client: httpClient}
}

func (client *zbClient) GetName() string {
    return "ZB"
}

func (client *zbClient) decodeResponse(respByte []byte, respJSON zbCommonResponseProvider) error {
    if err := json.Unmarshal(respByte, respJSON); err != nil {
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
    respBytes, err := client.Get(zbBaseApi+"kline", http.WithQuery(map[string]string{
        "market": symbol,
        "type":   period,
        "size":   strconv.Itoa(size),
    }))
    if err != nil {
        return 0, err
    }

    var respJSON zbKlineResponse
    err = client.decodeResponse(respBytes, &respJSON)
    if err != nil {
        return 0, err
    }
    logrus.Debugf("%s - Kline for %s*%v uses price at %s", client.GetName(), period, size,
        time.Unix(int64(respJSON.Data[0][0])/1000, 0))
    return respJSON.Data[0][1], nil
}

func (client *zbClient) GetSymbolPrice(symbol string) (*SymbolPrice, error) {
    respBytes, err := client.Get(zbBaseApi+"ticker", http.WithQuery(map[string]string{"market": strings.ToLower(symbol)}))
    if err != nil {
        return nil, err
    }

    var respJSON zbTickerResponse
    err = client.decodeResponse(respBytes, &respJSON)
    if err != nil {
        return nil, err
    }

    var percentChange1h, percentChange24h = math.MaxFloat64, math.MaxFloat64
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

func init() {
    Register(NewZBClient)
}
