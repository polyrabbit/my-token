package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/polyrabbit/token-ticker/exchange/model"

	"github.com/mattn/go-colorable"
	"github.com/polyrabbit/token-ticker/writer"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Will be set by go-build
var (
	Version       string
	Rev           string
	exampleConfig string
)

func init() {
	// Set log format
	formatter := &logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "15:04:05",
	}
	logrus.SetFormatter(formatter)
	logrus.SetOutput(colorable.NewColorableStderr()) // For Windows

	showVersion := pflag.BoolP("Version", "v", false, "Show Version number")
	showHelp := pflag.BoolP("help", "h", false, "Show usage message")
	pflag.CommandLine.MarkHidden("help")
	pflag.BoolP("debug", "d", false, "Enable debug mode")
	showExchanges := pflag.BoolP("list-exchanges", "l", false, "List supported exchanges")
	pflag.IntP("refresh", "r", 0, "Auto refresh on every specified seconds, "+
		"note every exchange has a rate limit, \ntoo frequent refresh may cause your IP banned by their servers")

	var configFile string
	pflag.StringVarP(&configFile, "config-file", "c", "", `Config file path, use "--example-config-file <path>" `+
		"to generate an example config file,\n"+
		"by default token-ticker uses \"token_ticker.yml\" in current directory or $HOME as config file")
	var exampleConfigFile string
	pflag.StringVar(&exampleConfigFile, "example-config-file", "",
		"Generate example config file to the specified file path, by default it outputs to stdout")
	pflag.Lookup("example-config-file").NoOptDefVal = "-"

	pflag.StringSliceP("show", "s", writer.GetColumns(), "Only show comma-separated columns")
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

	if *showExchanges {
		fmt.Fprintln(os.Stderr, "Supported exchanges:")
		for _, name := range model.GetAllNames() {
			fmt.Fprintf(os.Stderr, " %s\n", name)
		}
		os.Exit(0)
	}

	if exampleConfigFile != "" {
		writeExampleConfig(exampleConfigFile)
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
			logrus.Warnf("Error reading config file: %v", err)
		}
	}
	if viper.GetBool("debug") {
		logrus.SetLevel(logrus.DebugLevel)
	}
	logrus.Debugln("Using config file:", viper.ConfigFileUsed())
}

func showUsageAndExit() {
	// Print usage message and exit
	fmt.Fprintf(os.Stderr, "\nUsage: %s [Options] [Exchange1.Token1 Exchange2.Token2 ...]\n", os.Args[0])
	fmt.Fprintln(os.Stderr, "\nTrack token prices of your favorite exchanges in the terminal")
	fmt.Fprintln(os.Stderr, "\nOptions:")
	pflag.PrintDefaults()
	fmt.Fprintln(os.Stderr, "\nExchange.Token Pairs:")
	fmt.Fprintln(os.Stderr, "  Specify which exchange and token pair to query, different exchanges use different forms to express tokens/trading pairs, refer to their URLs to find the format"+
		" (eg. to get BitCoin price from Bitfinex and CoinMarketCap you should use query string \"Bitfinex.BTCUSDT CoinMarketCap.Bitcoin\").")
	fmt.Fprintln(os.Stderr, "\nFind help/updates from here - https://github.com/polyrabbit/token-ticker")
	os.Exit(0)
}

func writeExampleConfig(fpath string) {
	if exampleConfig == "" {
		logrus.Fatalln("example config should be set by build script!")
	}
	fout, err := os.Stdout, error(nil)
	if fpath != "-" {
		if _, err := os.Stat(fpath); err == nil {
			logrus.Warnf("%s already exists, skipping", fpath)
			return
		}
		if fout, err = os.Create(fpath); err != nil {
			logrus.Errorf("Failed to create config file %s, error: %v", fpath, err)
			return
		}
		defer fout.Close()
	}
	if _, err = fout.WriteString(exampleConfig); err != nil {
		logrus.Errorf("Failed to write config file %s, error: %v", fpath, err)
	} else if fout != os.Stdout {
		logrus.Infof("Write example config file to %s", fpath)
	}
}

func parseQueryFromCLI(cliArgs []string) []*model.PriceQuery {
	var (
		lastExchangeDef = &model.PriceQuery{}
		exchangeList    []*model.PriceQuery
	)
	for _, arg := range cliArgs {
		tokenDef := strings.SplitN(arg, ".", 2)
		if len(tokenDef) != 2 {
			logrus.Fatalf("Unrecognized token definition - %s, expecting {exchange}.{token}", arg)
		}
		if lastExchangeDef.Name == tokenDef[0] {
			// Merge consecutive exchange definitions
			// Do not sort/reorder here, to remain the order user specified
			lastExchangeDef.Tokens = append(lastExchangeDef.Tokens, tokenDef[1])
		} else {
			exchangeDef := &model.PriceQuery{
				Name:   tokenDef[0],
				Tokens: []string{tokenDef[1]}}
			lastExchangeDef = exchangeDef
			exchangeList = append(exchangeList, exchangeDef)
		}
	}
	return exchangeList
}

func MustParsePriceQueries() []*model.PriceQuery {
	if pflag.NArg() != 0 {
		// Construct exchange from command-line
		return parseQueryFromCLI(pflag.Args())
	}
	// Read from config file
	var queries []*model.PriceQuery
	err := viper.UnmarshalKey("exchanges", &queries)
	if err != nil {
		logrus.Fatalf("Unable to decode config file, %v", err)
	}
	return queries
}