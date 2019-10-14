package model

import (
	"sort"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
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

var (
	exchangeRegistry      = make(map[string]ExchangeClient)
	officialExchangeNames []string
)

func Register(client ExchangeClient) {
	name := client.GetName()
	officialExchangeNames = append(officialExchangeNames, name)
	exchangeRegistry[strings.ToUpper(name)] = client
}

// Factory method to create exchange client
func getExchangeClient(exchangeName string) ExchangeClient {
	exchangeName = strings.ToUpper(exchangeName)
	if client, ok := exchangeRegistry[exchangeName]; ok {
		return client
	}
	return nil
}

func GetAllNames() []string {
	sort.Strings(officialExchangeNames)
	return officialExchangeNames
}

type PriceQuery struct {
	Name   string
	Tokens []string
}

// Return a slice of waiting chans, each of them represents a pending request
func getPricesAsync(client ExchangeClient, symbols []string) []chan *SymbolPrice {
	// Use slice to hold the waiting chans in order to keep requested order
	waitingChans := make([]chan *SymbolPrice, 0, len(symbols))
	for _, symbol := range symbols {
		doneCh := make(chan *SymbolPrice, 1)
		waitingChans = append(waitingChans, doneCh)
		go func(symbol string) {
			sp, err := client.GetSymbolPrice(symbol)
			if err != nil {
				logrus.Warnf("Failed to get symbol price for %s from %s, error: %s", symbol, client.GetName(), err)
				if strings.Contains(err.Error(), "i/o timeout") {
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

func GetSymbolPrices(priceQueries []*PriceQuery) []*SymbolPrice {
	// Loop all priceQueries from config
	waitingChanList := make([]chan *SymbolPrice, 0, len(priceQueries))
	for _, query := range priceQueries {
		client := getExchangeClient(query.Name)
		if client == nil {
			logrus.Warnf("Unknown exchange %s, skipping", query.Name)
			continue
		}
		pendings := getPricesAsync(client, query.Tokens)
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
