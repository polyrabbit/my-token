package exchange

import (
	"github.com/bitfinexcom/bitfinex-api-go/v1"
	"net/http"
	"strings"
)

type bitfinixClient struct {
	innerClient *bitfinex.Client
}

func NewBitfinixClient(httpClient *http.Client) *bitfinixClient {
	http.DefaultClient = httpClient // luckily bitfinex uses the DefaultClient, override it here
	client := bitfinex.NewClient()
	return &bitfinixClient{innerClient: client}
}

func (client *bitfinixClient) GetName() string {
	return "Bitfinix"
}

func (client *bitfinixClient) GetSymbolPrice(symbol string) (*SymbolPrice, error) {
	ticker, err := client.innerClient.Ticker.Get(symbol)
	if err != nil {
		return nil, err
	}

	t, e := ticker.ParseTime()
	if e != nil {
		return nil, e
	}
	return &SymbolPrice{
		Symbol:   strings.ToUpper(symbol),
		Price:    ticker.LastPrice,
		Source:   client.GetName(),
		UpdateAt: *t,
		// Other fields are not supported, use api v2 to get them
	}, nil
}
