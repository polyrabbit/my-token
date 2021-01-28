package model

import (
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

type PriceQuery struct {
	Name   string
	Tokens []string
}
