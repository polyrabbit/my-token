package main

import (
	"encoding/json"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/mattn/go-colorable"
	"github.com/polyrabbit/my-token/config"
	"github.com/polyrabbit/my-token/exchange"
	"github.com/polyrabbit/my-token/http"
	"github.com/polyrabbit/my-token/writer"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func checkForUpdate(httpClient *http.Client) {
	const releaseURL = "https://api.github.com/repos/polyrabbit/my-token/releases/latest"
	respBytes, err := httpClient.Get(releaseURL, nil)
	if err != nil {
		logrus.Debugf("Failed to fetch Github release page, error %v", err)
		return
	}

	var releaseJSON struct {
		Tag string `json:"tag_name"`
		URL string `json:"html_url"`
	}
	if err := json.Unmarshal(respBytes, &releaseJSON); err != nil {
		logrus.Debugf("Failed to decode Github release page JSON, error %v", err)
		return
	}
	releaseJSON.Tag = strings.TrimPrefix(releaseJSON.Tag, "v")
	logrus.Debugf("Latest release tag is %s", releaseJSON.Tag)
	if config.Version != "" && releaseJSON.Tag != config.Version {
		color.New(color.FgYellow).Fprintf(os.Stderr,
			"my-token %s is available (you're using %s), get the latest release from: %s\n",
			releaseJSON.Tag, config.Version, releaseJSON.URL)
	}
}

func main() {
	config.Parse()
	httpClient := http.New()
	registry := exchange.NewRegistry(httpClient)
	if viper.GetBool("list-exchanges") {
		config.ListExchangesAndExit(registry.GetAllNames())
	}
	go checkForUpdate(httpClient)

	refreshInterval := viper.GetInt("refresh")
	if refreshInterval != 0 {
		logrus.Infof("Auto refresh on every %d seconds", refreshInterval)
	}

	var tableWriter = writer.NewTableWriter()
	logrus.SetOutput(tableWriter)
	defer logrus.SetOutput(colorable.NewColorableStderr())

	queries := config.MustParsePriceQueries()
	for {
		symbolPriceList := registry.GetSymbolPrices(queries)
		tableWriter.Render(symbolPriceList)
		if refreshInterval == 0 {
			break
		}
		// Use sleep here so I can stall as much as I can to avoid exceeding API limit
		time.Sleep(time.Duration(refreshInterval) * time.Second)
	}
}
