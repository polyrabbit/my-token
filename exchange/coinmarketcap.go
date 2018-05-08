package exchange

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// https://coinmarketcap.com/api/
const coinmarketcapBaseApi = "https://api.coinmarketcap.com/v1/ticker/"

type coinMarketCapClient struct {
	exchangeBaseClient
	AccessKey string
	SecretKey string
}

type coinMarketCapToken struct {
	ID               string
	Name             string
	Symbol           string
	Rank             int32   `json:",string"`
	PriceUSD         string  `json:"price_usd"`
	PriceBTC         float64 `json:"price_btc,string"`
	Volume24hUSD     float64 `json:"24h_volume_usd,string"`
	MarketCapUSD     float64 `json:"market_cap_usd,string"`
	AvailableSupply  float64 `json:"available_supply,string"`
	TotalSupply      float64 `json:"total_supply,string"`
	MaxSupply        float64 `json:"max_supply,string"`
	PercentChange1h  float64 `json:"percent_change_1h,string"`
	PercentChange24h float64 `json:"percent_change_24h,string"`
	PercentChange7d  float64 `json:"percent_change_7d,string"`
	LastUpdated      int64   `json:"last_updated,string"`
}

type notFoundResponse struct {
	Error string
}

// I don't like returning a general type here, any other better way to use the factory pattern?
func NewCoinmarketcapClient(httpClient *http.Client) *coinMarketCapClient {
	return &coinMarketCapClient{exchangeBaseClient: *newExchangeBase(coinmarketcapBaseApi, httpClient)}
}

func (client *coinMarketCapClient) GetName() string {
	return "CoinMarketCap"
}

func (client *coinMarketCapClient) GetSymbolPrice(symbol string) (*SymbolPrice, error) {
	resp, err := client.httpGet(symbol+"/", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)

	if resp.StatusCode == 404 {
		resp := &notFoundResponse{}
		if err := decoder.Decode(resp); err != nil {
			return nil, err
		}
		return nil, errors.New(resp.Error)
	}

	var tokens []coinMarketCapToken
	if err := decoder.Decode(&tokens); err != nil {
		return nil, err
	}

	if len(tokens) == 0 {
		return nil, fmt.Errorf("Cannot find symbol %s, got zero-sized array response", symbol)
	}
	token := tokens[0]

	return &SymbolPrice{
		Symbol:           token.Symbol,
		Price:            token.PriceUSD,
		Source:           client.GetName(),
		UpdateAt:         time.Unix(token.LastUpdated, 0),
		PercentChange1h:  token.PercentChange1h,
		PercentChange24h: token.PercentChange24h}, nil
}

func init() {
	register((&coinMarketCapClient{}).GetName(), func(client *http.Client) ExchangeClient {
		// Limited by type system in Go, I hate wrapper/adapter
		return NewCoinmarketcapClient(client)
	})
}
