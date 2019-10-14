package exchange

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/polyrabbit/token-ticker/exchange/model"

	"github.com/polyrabbit/token-ticker/http"
)

// https://coinmarketcap.com/api/
const coinmarketcapBaseApi = "https://api.coinmarketcap.com/v1/ticker/"

type coinMarketCapClient struct {
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

func (client *coinMarketCapClient) GetName() string {
	return "CoinMarketCap"
}

func (client *coinMarketCapClient) GetSymbolPrice(symbol string) (*model.SymbolPrice, error) {
	respBytes, err := http.Get(coinmarketcapBaseApi+symbol+"/", nil)
	if err != nil {
		if herr, ok := err.(*http.HTTPError); ok {
			resp := &notFoundResponse{}
			if err := json.Unmarshal(herr.Body, resp); err != nil {
				return nil, err
			}
			return nil, errors.New(resp.Error)
		}
		return nil, err
	}

	var tokens []coinMarketCapToken
	if err := json.Unmarshal(respBytes, &tokens); err != nil {
		return nil, err
	}

	if len(tokens) == 0 {
		return nil, fmt.Errorf("cannot find symbol %s, got zero-sized array response", symbol)
	}
	token := tokens[0]

	return &model.SymbolPrice{
		Symbol:           token.Symbol,
		Price:            token.PriceUSD,
		Source:           client.GetName(),
		UpdateAt:         time.Unix(token.LastUpdated, 0),
		PercentChange1h:  token.PercentChange1h,
		PercentChange24h: token.PercentChange24h}, nil
}

func init() {
	model.Register(new(coinMarketCapClient))
}
