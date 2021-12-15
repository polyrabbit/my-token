package config

import "strings"

const (
	ColumnSymbol       = "Symbol"
	ColumnPrice        = "Price"
	ColumnChange1hPct  = "%Change(1h)"
	ColumnChange24hPct = "%Change(24h)"
	ColumnSource       = "Source"
	ColumnUpdated      = "Updated"
)

func supportedColumns() []string {
	return []string{ColumnSymbol, ColumnPrice, ColumnChange1hPct, ColumnChange24hPct, ColumnSource, ColumnUpdated}
}

type PriceQuery struct {
	Name   string   `mapstructure:"name"`
	Tokens []string `mapstructure:"tokens"`
	APIKey string   `mapstructure:"api_key"`
}

type Config struct {
	Timeout int           `mapstructure:"timeout"`
	Proxy   string        `mapstructure:"proxy"`
	Refresh int           `mapstructure:"refresh"`
	Columns []string      `mapstructure:"show"`
	Debug   bool          `mapstructure:"debug"`
	Queries []*PriceQuery `mapstructure:"exchanges"`
}

func (c *Config) GroupQueryByExchange() map[string]PriceQuery {
	exchangeMap := make(map[string]PriceQuery, len(c.Queries))
	for _, query := range c.Queries {
		exchangeMap[strings.ToUpper(query.Name)] = *query
	}
	return exchangeMap
}
