package exchange

import (
	"errors"
	"fmt"
	"strings"

	"github.com/polyrabbit/my-token/config"
	"github.com/polyrabbit/my-token/http"
	"github.com/tidwall/gjson"
)

// https://coinmarketcap.com/api/documentation/v1/#operation/getV1CryptocurrencyQuotesLatest
const coinmarketcapBaseApi = "https://pro-api.coinmarketcap.com"

type coinMarketCapClient struct {
	*http.Client
	APIKey string
}

// An example 200 response
// {
//     "status": {
//         "timestamp": "2021-02-20T13:18:48.729Z",
//         "error_code": 0,
//         "error_message": null,
//         "elapsed": 17,
//         "credit_count": 1,
//         "notice": null
//     },
//     "data": {
//         "BTC": {
//             "id": 1,
//             "name": "Bitcoin",
//             "symbol": "BTC",
//             "slug": "bitcoin",
//             "num_market_pairs": 9713,
//             "date_added": "2013-04-28T00:00:00.000Z",
//             "tags": [
//                 "mineable",
//                 "pow",
//                 "sha-256",
//                 "store-of-value",
//                 "state-channels",
//                 "coinbase-ventures-portfolio",
//                 "three-arrows-capital-portfolio",
//                 "polychain-capital-portfolio"
//             ],
//             "max_supply": 21000000,
//             "circulating_supply": 18633843,
//             "total_supply": 18633843,
//             "is_active": 1,
//             "platform": null,
//             "cmc_rank": 1,
//             "is_fiat": 0,
//             "last_updated": "2021-02-20T13:17:02.000Z",
//             "quote": {
//                 "USD": {
//                     "price": 57088.920608781234,
//                     "volume_24h": 65358403259.270164,
//                     "percent_change_1h": 2.15396551,
//                     "percent_change_24h": 8.31814582,
//                     "percent_change_7d": 21.8879362,
//                     "percent_change_30d": 81.30631608,
//                     "market_cap": 1063785983663.4939,
//                     "last_updated": "2021-02-20T13:17:02.000Z"
//                 }
//             }
//         }
//     }
// }

func NewCoinMarketCapClient(queries map[string]config.PriceQuery, httpClient *http.Client) ExchangeClient {
	c := &coinMarketCapClient{Client: httpClient}
	if query, ok := queries[strings.ToUpper(c.GetName())]; ok { // If user queries CoinMarketCap, then API key is required
		c.APIKey = query.APIKey
		if c.APIKey == "" {
			panic(fmt.Errorf("%s now requires API key, get one from https://coinmarketcap.com/api/", c.GetName()))
		}
	}
	return c
}

func (client *coinMarketCapClient) GetName() string {
	return "CoinMarketCap"
}

func (client *coinMarketCapClient) HTTPHeader() map[string]string {
	return map[string]string{
		"X-CMC_PRO_API_KEY": client.APIKey,
	}
}

func (client *coinMarketCapClient) GetSymbolPrice(symbol string) (*SymbolPrice, error) {
	respBytes, err := client.Get(coinmarketcapBaseApi+"/v1/cryptocurrency/quotes/latest",
		http.WithQuery(map[string]string{"symbol": strings.ToUpper(symbol)}),
		http.WithHeader(client.HTTPHeader()))
	// If there is a more specific error
	if errMsg := gjson.GetBytes(respBytes, "status.error_message"); errMsg.String() != "" {
		return nil, errors.New(errMsg.String())
	}
	// Then throws a generic one
	if err != nil {
		return nil, err
	}

	symbolInfo := gjson.GetBytes(respBytes, fmt.Sprintf("data.%s", strings.ToUpper(symbol)))
	if !symbolInfo.Exists() {
		return nil, fmt.Errorf("no symbol %q found in returned map", symbol)
	}
	usdQuote := gjson.GetBytes([]byte(symbolInfo.Raw), "quote.USD")
	if !usdQuote.Exists() {
		return nil, fmt.Errorf("quote.USD not found in %q", symbol)
	}

	return &SymbolPrice{
		Symbol:           symbolInfo.Get("symbol").String(),
		Price:            usdQuote.Get("price").String(),
		Source:           client.GetName(),
		UpdateAt:         usdQuote.Get("last_updated").Time(),
		PercentChange1h:  usdQuote.Get("percent_change_1h").Float(),
		PercentChange24h: usdQuote.Get("percent_change_24h").Float()}, nil
}

func init() {
	Register(NewCoinMarketCapClient)
}
