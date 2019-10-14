package exchange

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/polyrabbit/token-ticker/exchange/model"

	"github.com/buger/jsonparser"
	"github.com/polyrabbit/token-ticker/http"
	"github.com/sirupsen/logrus"
)

// https://www.kraken.com/help/api
const krakenBaseApi = "https://api.kraken.com/0/public/"

type krakenClient struct {
	AccessKey string
	SecretKey string
}

func (client *krakenClient) GetName() string {
	return "Kraken"
}

/**
Read response and check any potential errors
*/
func (client *krakenClient) readResponse(respBytes []byte) ([]byte, error) {
	var errorMsg []string
	jsonparser.ArrayEach(respBytes, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
		if dataType == jsonparser.String {
			errorMsg = append(errorMsg, string(value))
		}
	}, "error")
	if len(errorMsg) != 0 {
		return nil, errors.New(strings.Join(errorMsg, ", "))
	}
	return respBytes, nil
}

func (client *krakenClient) GetKlinePrice(symbol string, since time.Time, interval int) (float64, error) {
	symbolUpperCase := strings.ToUpper(symbol)
	respByte, err := http.Get(krakenBaseApi+"OHLC", map[string]string{
		"pair":     symbolUpperCase,
		"since":    strconv.FormatInt(since.Unix(), 10),
		"interval": strconv.Itoa(interval),
	})
	if err != nil {
		return 0, err
	}

	content, err := client.readResponse(respByte)
	if err != nil {
		return 0, err
	}
	// jsonparser saved my life, no need to struggle with different/weird response types
	klineBytes, dataType, _, err := jsonparser.Get(content, "result", symbolUpperCase, "[0]")
	if err != nil {
		return 0, err
	}
	if dataType != jsonparser.Array {
		return 0, fmt.Errorf("kline should be an array, getting %s", dataType)
	}

	timestamp, err := jsonparser.GetInt(klineBytes, "[0]")
	if err != nil {
		return 0, err
	}
	openPrice, err := jsonparser.GetString(klineBytes, "[1]")
	if err != nil {
		return 0, err
	}
	logrus.Debugf("%s - Kline for %s uses open price at %s", client.GetName(), since.Local(),
		time.Unix(timestamp, 0).Local())
	return strconv.ParseFloat(openPrice, 64)
}

func (client *krakenClient) GetSymbolPrice(symbol string) (*model.SymbolPrice, error) {
	respByte, err := http.Get(krakenBaseApi+"Ticker", map[string]string{"pair": strings.ToUpper(symbol)})
	if err != nil {
		return nil, err
	}

	content, err := client.readResponse(respByte)
	if err != nil {
		return nil, err
	}

	lastPriceString, err := jsonparser.GetString(content, "result", strings.ToUpper(symbol), "c", "[0]")
	if err != nil {
		return nil, err
	}
	lastPrice, err := strconv.ParseFloat(lastPriceString, 64)
	if err != nil {
		return nil, err
	}

	time.Sleep(time.Second) // API call rate limit
	var (
		now              = time.Now()
		percentChange1h  = math.MaxFloat64
		percentChange24h = math.MaxFloat64
	)
	price1hAgo, err := client.GetKlinePrice(symbol, now.Add(-61*time.Minute), 1)
	if err != nil {
		logrus.Warnf("%s - Failed to get price 1 hour ago, error: %v\n", client.GetName(), err)
	} else if price1hAgo != 0 {
		percentChange1h = (lastPrice - price1hAgo) / price1hAgo * 100
	}
	price24hAgo, err := client.GetKlinePrice(symbol, now.Add(-24*time.Hour), 5)
	if err != nil {
		logrus.Warnf("%s - Failed to get price 24 hours ago, error: %v\n", client.GetName(), err)
	} else if price24hAgo != 0 {
		percentChange24h = (lastPrice - price24hAgo) / price24hAgo * 100
	}

	return &model.SymbolPrice{
		Symbol:           symbol,
		Price:            lastPriceString,
		UpdateAt:         time.Now(),
		Source:           client.GetName(),
		PercentChange1h:  percentChange1h,
		PercentChange24h: percentChange24h,
	}, nil
}

func init() {
	model.Register(new(krakenClient))
}
