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

// https://github.com/binance-exchange/binance-official-api-docs/blob/master/rest-api.md
const binanceBaseApi = "https://api.binance.com"

type binanceClient struct {
	*http.Client
}

type binanceErrorResponse struct {
	Code int32
	Msg  *string
}

type binancePriceResponse struct {
	binanceErrorResponse
	Symbol string
	Price  string
}

type binance24hStatistics struct {
	binanceErrorResponse
	Symbol             string
	LastPrice          string
	PrevClosePrice     string
	PriceChange        float64 `json:",string"`
	PriceChangePercent float64 `json:",string"`
	OpenTime           int64
	CloseTime          int64
}

func NewBinanceClient(queries map[string]*config.PriceQuery, httpClient *http.Client) ExchangeClient {
	return &binanceClient{Client: httpClient}
}

func (client *binanceClient) GetName() string {
	return "Binance"
}

func (client *binanceClient) GetPrice1hAgo(symbol string) (float64, error) {
	now := time.Now()
	lastHour := now.Add(-1 * time.Hour)
	respBytes, err := client.Get(binanceBaseApi+"/api/v1/klines", http.WithQuery(map[string]string{
		"symbol":    strings.ToUpper(symbol),
		"interval":  "1m",
		"limit":     "1",
		"startTime": strconv.FormatInt(lastHour.Unix()*1000, 10),
	}))
	if err != nil {
		return 0, err
	}

	var klines [][]interface{}
	if err := json.Unmarshal(respBytes, &klines); err != nil {
		return 0, err
	}
	if len(klines) == 0 {
		return 0, errors.New("got an empty kline")
	}
	if s, ok := klines[0][1].(string); ok {
		p, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0, fmt.Errorf("failed to convert %v to float", s)
		}
		return p, nil
	}
	return 0, fmt.Errorf("failed to convert %v to string", klines[0][1])
}

func (client *binanceClient) Get24hStatistics(symbol string) (*binance24hStatistics, error) {
	// always return an empty response, so the caller doesn't need to handle error
	var respJSON binance24hStatistics

	respBytes, err := client.Get(binanceBaseApi+"/api/v1/ticker/24hr", http.WithQuery(map[string]string{"symbol": strings.ToUpper(symbol)}))
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(respBytes, &respJSON); err != nil {
		return nil, err
	}

	if respJSON.Msg != nil {
		return nil, errors.New(*respJSON.Msg)
	}
	return &respJSON, nil
}

func (client *binanceClient) GetSymbolPrice(symbol string) (*SymbolPrice, error) {
	// I found 24 hour price statistics already covers required info, uncomment the following code if needed

	//rawUrl := client.buildUrl("/api/v3/ticker/price", map[string]string{"symbol": strings.ToUpper(symbol)})
	//resp, err := client.HTTPClient.Get(rawUrl)
	//if err != nil {
	//	return nil, err
	//}
	//defer resp.Body.Close()
	//
	//decoder := json.NewDecoder(resp.Body)
	//var respJSON binancePriceResponse
	//if err := decoder.Decode(&respJSON); err != nil {
	//	return nil, err
	//}
	//
	//if respJSON.Msg != nil {
	//	return nil, errors.New(*respJSON.Msg)
	//}

	stat24h, err := client.Get24hStatistics(symbol)
	if err != nil {
		//logrus.Warnf("Failed to get 24 hour price change statistics, error: %s\n", err)
		return nil, err
	}

	var percentChange1h = math.MaxFloat64
	price1hAgo, err2 := client.GetPrice1hAgo(symbol)
	if err2 != nil {
		logrus.Warnf("%s - Failed on GetPrice1hAgo, error: %s\n", client.GetName(), err2)
	} else if price1hAgo != 0 {
		currentPrice, err := strconv.ParseFloat(stat24h.LastPrice, 64)
		if err != nil {
			logrus.Warnf("%s - Failed to convert current price %v to float", client.GetName(), stat24h.LastPrice)
		}
		percentChange1h = (currentPrice - price1hAgo) / price1hAgo * 100
	}

	return &SymbolPrice{
		Symbol:           symbol,
		Price:            stat24h.LastPrice,
		UpdateAt:         time.Unix(stat24h.CloseTime/1000, 0),
		Source:           client.GetName(),
		PercentChange1h:  percentChange1h,
		PercentChange24h: stat24h.PriceChangePercent,
	}, nil
}

func init() {
	Register(NewBinanceClient)
}
