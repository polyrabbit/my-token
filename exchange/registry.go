package exchange

import (
	"fmt"
	"net"
	"sort"
	"strings"
	"time"

	"github.com/polyrabbit/my-token/config"
	"github.com/polyrabbit/my-token/http"
	"github.com/sirupsen/logrus"
)

type ExchangeClient interface {
	GetName() string
	GetSymbolPrice(string) (*SymbolPrice, error)
}

type ExchangeClientProvider func(queries map[string]config.PriceQuery, httpClient *http.Client) ExchangeClient

var providers []ExchangeClientProvider

func Register(p ExchangeClientProvider) {
	providers = append(providers, p)
}

type Registry struct {
	clients       map[string]ExchangeClient
	officialNames []string
	hasProxy      bool
}

func NewRegistry(cfg *config.Config, httpClient *http.Client) *Registry {
	exchangeMap := cfg.GroupQueryByExchange()
	r := &Registry{clients: make(map[string]ExchangeClient), hasProxy: cfg.Proxy != ""}
	for _, p := range providers {
		eClient := p(exchangeMap, httpClient)
		r.officialNames = append(r.officialNames, eClient.GetName())
		upperName := strings.ToUpper(eClient.GetName())
		if _, exist := r.clients[upperName]; exist {
			panic(fmt.Errorf("%q already exists in exchange registry", upperName))
		}
		r.clients[upperName] = eClient
	}
	return r
}

func (r *Registry) GetAllNames() []string {
	sort.Strings(r.officialNames)
	return r.officialNames
}

func (r *Registry) GetSymbolPrices(priceQueries []*config.PriceQuery) []*SymbolPrice {
	// Loop all priceQueries from config
	waitingChanList := make([]chan *SymbolPrice, 0, len(priceQueries))
	for _, query := range priceQueries {
		client := r.getClient(query.Name)
		if client == nil {
			logrus.Warnf("Unknown exchange %s, skipping", query.Name)
			continue
		}
		pendings := r.getPricesAsync(client, query.Tokens)
		waitingChanList = append(waitingChanList, pendings...)
	}

	symbolPriceList := make([]*SymbolPrice, 0, len(waitingChanList))
	for _, doneCh := range waitingChanList {
		sp := <-doneCh
		if sp != nil {
			symbolPriceList = append(symbolPriceList, sp)
		}
	}
	return symbolPriceList
}

// Factory method to create exchange client
func (r *Registry) getClient(exchangeName string) ExchangeClient {
	exchangeName = strings.ToUpper(exchangeName)
	if client, ok := r.clients[exchangeName]; ok {
		return client
	}
	return nil
}

// Return a slice of waiting chans, each of them represents a pending request
func (r *Registry) getPricesAsync(client ExchangeClient, symbols []string) []chan *SymbolPrice {
	// Use slice to hold the waiting chans in order to keep requested order
	waitingChans := make([]chan *SymbolPrice, 0, len(symbols))
	for _, symbol := range symbols {
		doneCh := make(chan *SymbolPrice, 1)
		waitingChans = append(waitingChans, doneCh)
		go func(symbol string) {
			start := time.Now()
			sp, err := client.GetSymbolPrice(symbol)
			if err != nil {
				logEntry := logrus.WithError(err)
				e, ok := err.(net.Error)
				if ok && e.Timeout() {
					elapsed := time.Since(start)
					logEntry = logEntry.WithField("elapsed", elapsed.String())
				}
				logEntry.Warnf("Failed to get symbol price for %s from %s", symbol, client.GetName())
				if r.hasProxy && ok && e.Timeout() {
					logrus.Info("Maybe you are blocked by a firewall, try using --proxy to go through a proxy?")
				}
				close(doneCh) // close channel to indicate an error has happened, any other good idea?
			} else {
				doneCh <- sp
			}
		}(symbol)
	}
	return waitingChans
}
