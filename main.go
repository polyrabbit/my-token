package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"encoding/json"
	"github.com/gosuri/uilive"
	"github.com/olekukonko/tablewriter"
	. "github.com/polyrabbit/token-ticker/exchange"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"math"
)

// Will be set by go-build
var Version = ""
var Rev = ""

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
				if strings.Contains(err.Error(), "i/o timeout") {
					logrus.Info("Maybe you are blocked by a firewall, try using --proxy to go through a proxy?")
				}
				close(done) // close channel to indicate an error has happened, any other good idea?
			} else {
				done <- sp
			}
		}(symbol)
	}
	return waitingChans
}

func checkForUpdate(httpClient *http.Client) {
	releaseUrl := "https://api.github.com/repos/polyrabbit/token-ticker/releases/latest"
	resp, err := httpClient.Get(releaseUrl)
	if err != nil {
		logrus.Debugf("Failed to fetch Github release page, error %s", err)
		return
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)
	var releaseJSON struct {
		Tag string `json:"tag_name"`
		Url string `json:"html_url"`
	}
	if err := decoder.Decode(&releaseJSON); err != nil {
		logrus.Debugf("Failed to decode Github release page JSON, error %s", err)
		return
	}
	if releaseJSON.Tag == "" {
		logrus.Debugf("Get an empty release tag?")
		return
	}
	releaseJSON.Tag = strings.TrimPrefix(releaseJSON.Tag, "v")
	logrus.Debugf("Latest release tag is %s", releaseJSON.Tag)
	if Version != "" && releaseJSON.Tag != Version {
		fmt.Fprintf(os.Stderr, "%s[%dmYou are using version %s, however version %s is available.\n",
			tablewriter.ESC, tablewriter.FgYellowColor, Version, releaseJSON.Tag)
		fmt.Fprintf(os.Stderr, "You should consider getting the latest release from '%s'.\n%s[%dm",
			releaseJSON.Url, tablewriter.ESC, tablewriter.Normal)
	}
}

func dimText(text string) string {
	const fontDim = 2
	return fmt.Sprintf("%s[%dm%s%s[%dm", tablewriter.ESC, fontDim, text, tablewriter.ESC, tablewriter.Normal)
}

func highlightChange(changePct float64) string {
	if changePct == math.MaxFloat64 {
		return ""
	}
	changeText := strconv.FormatFloat(changePct, 'f', 2, 64)
	if changePct == 0 {
		changeText = dimText("0")
	} else if changePct > 0 {
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
	fmt.Fprintf(os.Stderr, "\nUsage: %s [Options] [Token1 Token2 ...]\n", os.Args[0])
	fmt.Fprintln(os.Stderr, "\nTrack token prices of your favorite exchanges in the terminal")
	fmt.Fprintln(os.Stderr, "\nOptions:")
	pflag.PrintDefaults()
	fmt.Fprintln(os.Stderr, "\nTokens:")
	fmt.Fprintln(os.Stderr, "  Exchanges use many different forms to express tokens/symbols/currency pairs/markets, refer to their URLs to find the format"+
		"\n  eg. to get BitCoin price you should query Bitfinex using \"BTCUSDT\" and \"Bitcoin\" for CoinMarketCap")
	fmt.Fprintln(os.Stderr, "\nFind help/updates from here - https://github.com/polyrabbit/token-ticker")
	os.Exit(0)
}

func init() {
	// Set log format
	formatter := &logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "15:04:05",
	}
	logrus.SetFormatter(formatter)

	showVersion := pflag.BoolP("version", "v", false, "Show version number")
	showHelp := pflag.BoolP("help", "h", false, "Show usage message")
	pflag.CommandLine.MarkHidden("help")
	pflag.BoolP("debug", "d", false, "Enable debug mode")
	pflag.StringP("exchange", "e", "CoinMarketCap", "Source to get token price")
	showList := pflag.BoolP("list-exchanges", "l", false, "List supported exchanges")
	pflag.IntP("refresh", "r", 0, "Auto refresh on every specified seconds, "+
		"note every exchange has a rate limit, \ntoo frequent refresh may cause your IP banned by their servers")
	var configFile string
	pflag.StringVarP(&configFile, "config-file", "c", "", "Config file path, "+
		"refer to \"token_ticker.example.yaml\" for the format, \nby default token-ticker uses \"token_ticker.yml\" "+
		"in current directory or $HOME as config file")
	pflag.StringP("proxy", "p", "", "Proxy used when sending HTTP request \n(eg. "+
		"\"http://localhost:7777\", \"https://localhost:7777\", \"socks5://localhost:1080\")")
	pflag.IntP("timeout", "t", 20, "HTTP request timeout in seconds")
	pflag.CommandLine.SortFlags = false
	pflag.Usage = showUsageAndExit
	pflag.Parse()

	if *showHelp {
		showUsageAndExit()
	}

	if *showVersion {
		fmt.Fprintf(os.Stderr, "Version %s", Version)
		if Rev != "" {
			fmt.Fprintf(os.Stderr, ", build %s", Rev)
		}
		fmt.Fprintln(os.Stderr)
		os.Exit(0)
	}

	if *showList {
		fmt.Fprintln(os.Stderr, "Supported exchanges:")
		for _, name := range ListExchanges() {
			fmt.Fprintf(os.Stderr, " %s\n", name)
		}
		os.Exit(0)
	}

	viper.BindPFlags(pflag.CommandLine)
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
			if pflag.NArg() == 0 { // And no specified tokens
				showUsageAndExit()
			}
		default:
			logrus.Warnf("Error reading config file: %s\n", err)
		}
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
		var client = CreateExchangeClient(exchangeCfg.Name, httpClient)
		if client == nil {
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

	if pflag.NArg() != 0 {
		// Construct exchange from command-line
		configs = append(configs, &exchangeConfig{
			Name:   viper.GetString("exchange"),
			Tokens: pflag.Args()})
	} else {
		// Read from config file
		err := viper.UnmarshalKey("exchanges", &configs)
		if err != nil {
			logrus.Fatalf("Unable to decode config file, %v", err)
		}
	}

	refreshInterval := viper.GetInt("refresh")
	if refreshInterval != 0 {
		logrus.Infof("Auto refresh on every %d seconds", refreshInterval)
	}

	httpClient := newHttpClient(viper.GetString("proxy"))

	go checkForUpdate(httpClient)

	var writer = uilive.New()
	logrus.SetOutput(writer)
	defer logrus.SetOutput(os.Stderr)

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
