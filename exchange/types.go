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

	if queryMap != nil {
		query := url.Values{}
		for k, v := range queryMap {
			query.Set(k, v)
		}
		baseUrl.RawQuery = query.Encode()
	}
	return baseUrl.String()
}

func (client *exchangeBaseClient) httpGet(endpoint string, queryMap map[string]string) (*http.Response, error) {
	req, err := http.NewRequest("GET", client.buildUrl(endpoint, queryMap), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; token-ticker; +https://github.com/polyrabbit/token-ticker)")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Add("Cache-Control", "no-store")
	req.Header.Add("Cache-Control", "must-revalidate")
	return client.HTTPClient.Do(req)
}

type exchangeBuilder func(*http.Client) ExchangeClient

var exchangeRegistry = make(map[string]exchangeBuilder)
var officialExchangeNames []string

func register(name string, builder exchangeBuilder) {
	officialExchangeNames = append(officialExchangeNames, name)
	exchangeRegistry[strings.ToUpper(name)] = builder
}

// Factory method to create exchange client
func CreateExchangeClient(exchangeName string, httpClient *http.Client) ExchangeClient {
	exchangeName = strings.ToUpper(exchangeName)
	builder, ok := exchangeRegistry[exchangeName]
	if ok {
		return builder(httpClient)
	}
	return nil

	// Following are more flexible
	//switch strings.ToUpper(exchangeName) {
	//case "BINANCE":
	//	return NewBinanceClient(httpClient)
	//case "COINMARKETCAP":
	//	return NewCoinmarketcapClient(httpClient)
	//case "BITFINEX":
	//	return NewBitfinixClient(httpClient)
	//case "HUOBI":
	//	return NewHuobiClient(httpClient)
	//case "ZB":
	//	return NewZBClient(httpClient)
	//}
	//return nil
}

func ListExchanges() []string {
	sort.Strings(officialExchangeNames)
	return officialExchangeNames
}
