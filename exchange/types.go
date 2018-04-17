package exchange

import (
	"github.com/sirupsen/logrus"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"
	"time"
)

type SymbolPrice struct {
	Symbol           string
	Price            string
	Source           string
	UpdateAt         time.Time
	PercentChange1h  float64
	PercentChange24h float64
}

type ExchangeClient interface {
	GetName() string
	GetSymbolPrice(string) (*SymbolPrice, error)
}

type exchangeBaseClient struct {
	BaseURL    *url.URL
	HTTPClient *http.Client
}

func newExchangeBase(rawUrl string, httpClient *http.Client) *exchangeBaseClient {
	baseUrl, err := url.Parse(rawUrl)
	if err != nil {
		logrus.Fatalln(err)
	}
	return &exchangeBaseClient{baseUrl, httpClient}
}

func (client *exchangeBaseClient) buildUrl(endpoint string, queryMap map[string]string) string {
	baseUrl := *client.BaseURL
	baseUrl.Path = path.Join(baseUrl.Path, endpoint)

	query := url.Values{}
	for k, v := range queryMap {
		query.Set(k, v)
	}
	baseUrl.RawQuery = query.Encode()
	return baseUrl.String()
}

// Factory method to create exchange client
func CreateExchangeClient(exchangeName string, httpClient *http.Client) ExchangeClient {
	switch strings.ToUpper(exchangeName) {
	case "BINANCE":
		return NewBinanceClient(httpClient)
	case "COINMARKETCAP":
		return NewCoinmarketcapClient(httpClient)
	case "BITFINEX":
		return NewBitfinixClient(httpClient)
	}
	return nil
}

func ListExchanges() []string {
	supported := []string{"Binance", "CoinMarketCap", "Bitfinex"}
	sort.Strings(supported)
	return supported
}
