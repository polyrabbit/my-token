package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gosuri/uilive"
	"github.com/olekukonko/tablewriter"
	. "github.com/polyrabbit/token-ticker/exchange"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type ExchangeClient interface {
	GetName() string
	GetSymbolPrice(string) (*SymbolPrice, error)
}

type exchangeConfig struct {
	Name   string
	Tokens []string
}

// Return a slice of waiting chans, each of them represents a pending request
func requestSymbolPrice(client ExchangeClient, symbols []string) []chan *SymbolPrice {
	// Use slice to hold the waiting chans in order to keep requested order
	waitingChans := make([]chan *SymbolPrice, len(symbols))
	for i, symbol := range symbols {
		done := make(chan *SymbolPrice, 1)
		waitingChans[i] = done
		go func(symbol string) {
			sp, err := client.GetSymbolPrice(symbol)
			if err != nil {
				logrus.Warnf("Failed to get symbol price for %s from %s, error: %s", symbol, client.GetName(), err)
				close(done) // close channel to indicate an error has happened, any other good idea?
			} else {
				done <- sp
			}
		}(symbol)
	}
	return waitingChans
}

const fontDim = 2

func dimText(text string) string {
	return fmt.Sprintf("%s[%dm%s%s[%dm", tablewriter.ESC, fontDim, text, tablewriter.ESC, tablewriter.Normal)
}

func highlightChange(changePct float64) string {
	if changePct == 0 {
		return ""
	}
	changeText := strconv.FormatFloat(changePct, 'f', 2, 64)
	if changePct > 0 {
		changeText = fmt.Sprintf("%s[%dm%s%s[%dm", tablewriter.ESC, tablewriter.FgGreenColor,
			changeText, tablewriter.ESC, tablewriter.Normal)
	} else {
		changeText = fmt.Sprintf("%s[%dm%s%s[%dm", tablewriter.ESC, tablewriter.FgRedColor,
			changeText, tablewriter.ESC, tablewriter.Normal)
	}
	return changeText
}

func renderTable(symbolPriceList []*SymbolPrice, writer *uilive.Writer) {
	// Set up ascii table writer
	table := tablewriter.NewWriter(writer)
	headers := []string{"Symbol", "Price", "%Change (1h)", "%Change (24h)", "Source", "Updated"}
	table.SetHeader(headers)
	headerColors := make([]tablewriter.Colors, len(headers))
	for i, _ := range headerColors {
		headerColors[i] = tablewriter.Colors{tablewriter.FgYellowColor}
	}
	table.SetHeaderColor(headerColors...)
	table.SetRowLine(true)
	table.SetCenterSeparator(dimText("-"))
	table.SetColumnSeparator(dimText("|"))
	table.SetRowSeparator(dimText("-"))

	// Fill in data
	for _, sp := range symbolPriceList {
		table.Append([]string{sp.Symbol, sp.Price, highlightChange(sp.PercentChange1h),
			highlightChange(sp.PercentChange24h), sp.Source, sp.UpdateAt.Format("15:04:05")})
	}

	table.Render()
	writer.Flush()
}

func newHttpClient(rawProxyUrl string) *http.Client {
	transport := http.DefaultTransport.(*http.Transport)
	if rawProxyUrl != "" {
		proxyUrl, err := url.Parse(rawProxyUrl)
		if err != nil {
			logrus.Warnf("Failed to parse proxy URL: %s, error: %v, using system proxy\n", rawProxyUrl, err)
		} else {
			transportWithProxy := *transport                   // Copy the default transport
			transportWithProxy.Proxy = http.ProxyURL(proxyUrl) // Set custom proxy
			transport = &transportWithProxy
			logrus.Debugf("Using proxy %s", rawProxyUrl)
		}
	}

	logrus.Debugf("HTTP request timeout is set to %d", viper.GetInt("timeout"))
	return &http.Client{
		Transport: transport,
		Timeout:   time.Duration(viper.GetInt("timeout")) * time.Second,
	}
}

func showUsageAndExit() {
	// Print usage message and exit
	fmt.Fprintln(os.Stderr, "\nTrack token prices of your favorite exchanges in the terminal")
	fmt.Fprintln(os.Stderr, "\nOptions:")
	pflag.PrintDefaults()
	os.Exit(0)
}

func init() {
	// Set log format
	formatter := &logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "15:04:05",
	}
	logrus.SetFormatter(formatter)

	showHelp := pflag.BoolP("help", "h", false, "Show usage message")
	pflag.CommandLine.MarkHidden("help")
	pflag.BoolP("debug", "d", false, "Enable debug mode")
	pflag.IntP("refresh", "r", 0, "Auto refresh on every specified seconds, "+
		"note every exchange has a rate limit, \ntoo frequent refresh may cause your IP banned by their servers")
	var configFile string
	pflag.StringVarP(&configFile, "config-file", "c", "", "Config file path, "+
		"refer to \"token_ticker.example.yaml\" for the format, \nby default token-ticker uses \"token_ticker.yml\" "+
		"in current directory or $HOME as config file")
	pflag.StringP("proxy", "p", "", "Proxy used when sending HTTP request \nexample: "+
		"\"http://localhost:7777\", \"https://localhost:7777\", \"socks5://localhost:1080\"")
	pflag.Int("timeout", 0, "HTTP request timeout in seconds")
	pflag.CommandLine.SortFlags = false
	pflag.Usage = showUsageAndExit
	pflag.Parse()

	if *showHelp {
		showUsageAndExit()
	}

	viper.BindPFlags(pflag.CommandLine)
	viper.SetDefault("timeout", 20)
	// Set configure file
	viper.SetConfigName("token_ticker") // name of config file (without extension)
	viper.AddConfigPath(".")            // path to look for the config file in
	viper.AddConfigPath("$HOME")        // optionally look for config in the HOME directory
	viper.AddConfigPath("/etc")         // and /etc
	if configFile != "" {
		viper.SetConfigFile(configFile)
	}
	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {
		switch err.(type) {
		case viper.ConfigFileNotFoundError:
			showUsageAndExit()
		default:
			logrus.Warnf("Error reading config file: %s\n", err)
		}
		logrus.Warnf("Error reading config file: %s\n", err)
	}
	if viper.GetBool("debug") {
		logrus.SetLevel(logrus.DebugLevel)
	}
	logrus.Debugln("Using config file:", viper.ConfigFileUsed())
}

func getSymbolPrice(exchanges []*exchangeConfig, httpClient *http.Client) []*SymbolPrice {
	// Loop all exchanges from config
	var waitingChanList []chan *SymbolPrice
	for _, exchangeCfg := range exchanges {
		var client ExchangeClient
		switch strings.ToUpper(exchangeCfg.Name) {
		case "BINANCE":
			client = NewBinanceClient(httpClient)
		case "COINMARKETCAP":
			client = NewCoinmarketcapClient(httpClient)
		case "BITFINIX":
			client = NewBitfinixClient(httpClient)
		default:
			logrus.Warnf("Unknown exchange %s, skipping", exchangeCfg.Name)
			continue
		}
		pendings := requestSymbolPrice(client, exchangeCfg.Tokens)
		waitingChanList = append(waitingChanList, pendings...)
	}

	var symbolPriceList []*SymbolPrice
	for _, done := range waitingChanList {
		sp := <-done
		if sp != nil {
			symbolPriceList = append(symbolPriceList, sp)
		}
	}
	return symbolPriceList
}

func main() {
	var configs []*exchangeConfig
	err := viper.UnmarshalKey("exchanges", &configs)
	if err != nil {
		logrus.Fatalf("Unable to decode config file, %v", err)
	}

	refreshInterval := viper.GetInt("refresh")
	if refreshInterval != 0 {
		logrus.Infof("Auto refresh on every %d seconds", refreshInterval)
	}

	httpClient := newHttpClient(viper.GetString("proxy"))
	var writer = uilive.New()

	for {
		symbolPriceList := getSymbolPrice(configs, httpClient)
		renderTable(symbolPriceList, writer)
		if refreshInterval == 0 {
			break
		}
		// Use sleep here so I can stall as much as I can to avoid exceeding API limit
		time.Sleep(time.Duration(refreshInterval) * time.Second)
	}
}
