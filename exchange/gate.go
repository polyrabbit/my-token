package exchange

import (
    "encoding/json"
    "errors"
    "fmt"
    "math"
    "strconv"
    "strings"
    "time"

    "github.com/polyrabbit/my-token/config"
    "github.com/polyrabbit/my-token/http"
    "github.com/sirupsen/logrus"
)

// https://www.zb.com/i/developer
const gateBaseApi = "http://data.gateio.io/api2/1/"

type gateClient struct {
    *http.Client
}

type gateCommonResponse struct {
    Result  string
    Message *string
}

type gateTickerResponse struct {
    gateCommonResponse
    Last float64 `json:",string"`
}

type gateKlineResponse struct {
    gateCommonResponse
    Data [][]string
}

func (resp *gateTickerResponse) getCommonResponse() gateCommonResponse {
    return resp.gateCommonResponse
}

func (resp *gateKlineResponse) getCommonResponse() gateCommonResponse {
    return resp.gateCommonResponse
}

// Any way to hold the common response, instead of adding an interface here?
type gateCommonResponseProvider interface {
    getCommonResponse() gateCommonResponse
}

func NewGateClient(queries map[string]*config.PriceQuery, httpClient *http.Client) ExchangeClient {
    return &gateClient{Client: httpClient}
}

func (client *gateClient) GetName() string {
    return "Gate"
}

func (client *gateClient) decodeResponse(respBytes []byte, respJSON gateCommonResponseProvider) error {
    if err := json.Unmarshal(respBytes, respJSON); err != nil {
        return err
    }

    // All I need is to get the common part, I don't like this
    commonResponse := respJSON.getCommonResponse()
    if commonResponse.Message != nil {
        return errors.New(*commonResponse.Message)
    }
    return nil
}

func (client *gateClient) GetKlinePrice(symbol string, groupedSeconds int, size int) (float64, error) {
    symbol = strings.ToLower(symbol)
    respBytes, err := client.Get(gateBaseApi+"candlestick2/"+symbol, http.WithQuery(map[string]string{
        "group_sec":  strconv.Itoa(groupedSeconds),
        "range_hour": strconv.Itoa(size),
    }))
    if err != nil {
        return 0, err
    }

    var respJSON gateKlineResponse
    err = client.decodeResponse(respBytes, &respJSON)
    if err != nil {
        return 0, err
    }
    if len(respJSON.Data) == 0 {
        return 0, fmt.Errorf("%s - get a zero size kline response", client.GetName())
    }
    ts, err := strconv.ParseInt(respJSON.Data[0][0], 10, 64)
    if err != nil {
        return 0, err
    }
    logrus.Debugf("%s - Kline for %v hour(s) uses price at %s", client.GetName(), size,
        time.Unix(ts/1000, 0))
    return strconv.ParseFloat(respJSON.Data[0][5], 64)
}

func (client *gateClient) GetSymbolPrice(symbol string) (*SymbolPrice, error) {
    respBytes, err := client.Get(gateBaseApi + "ticker/" + symbol)
    if err != nil {
        return nil, err
    }

    var respJSON gateTickerResponse
    err = client.decodeResponse(respBytes, &respJSON)
    if err != nil {
        return nil, err
    }

    var percentChange1h, percentChange24h = math.MaxFloat64, math.MaxFloat64
    price1hAgo, err := client.GetKlinePrice(symbol, 60, 1)
    if err != nil {
        logrus.Warnf("%s - Failed to get price 1 hour ago, error: %v\n", client.GetName(), err)
    } else if price1hAgo != 0 {
        percentChange1h = (respJSON.Last - price1hAgo) / price1hAgo * 100
    }

    price24hAgo, err := client.GetKlinePrice(symbol, 300, 24) // Seems gate.io only supports 60, 300, 600 etc. seconds
    if err != nil {
        logrus.Warnf("%s - Failed to get price 24 hours ago, error: %v\n", client.GetName(), err)
    } else if price24hAgo != 0 {
        percentChange24h = (respJSON.Last - price24hAgo) / price24hAgo * 100
    }

    return &SymbolPrice{
        Symbol:           symbol,
        Price:            strconv.FormatFloat(respJSON.Last, 'f', -1, 64),
        UpdateAt:         time.Now(),
        Source:           client.GetName(),
        PercentChange1h:  percentChange1h,
        PercentChange24h: percentChange24h,
    }, nil
}

func init() {
    Register(NewGateClient)
}
