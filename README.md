# MyToken

[![Build Status](https://travis-ci.org/polyrabbit/my-token.svg?branch=master)](https://travis-ci.org/polyrabbit/my-token)
[![codecov](https://codecov.io/gh/polyrabbit/my-token/branch/master/graph/badge.svg)](https://codecov.io/gh/polyrabbit/my-token)
[![License: MIT](https://img.shields.io/badge/License-MIT-brightgreen.svg)](https://opensource.org/licenses/MIT)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](https://github.com/polyrabbit/my-token/pulls)
[![Go Report Card](https://goreportcard.com/badge/github.com/polyrabbit/my-token)](https://goreportcard.com/report/github.com/polyrabbit/my-token)

> NEVER LEAVE YOUR TERMINAL

![my-token](https://user-images.githubusercontent.com/2657334/76717485-8560d280-676e-11ea-94af-54a5e10e9b25.png)

my-token (or `mt` for short) is a CLI tool for those who are both **Crypto investors** and **Engineers**, allowing you to track token prices and changes in your favorite exchanges on the terminal.

### Features

 * Auto refresh on a specified interval, watch prices in live update mode
 * Proxy aware HTTP request, for easy access to blocked exchanges
 * Real-time prices from 12+ exchanges

### Supported Exchanges

 * [Binance](https://www.binance.com/)
 * [CoinMarketCap](https://coinmarketcap.com/)
 * [Bitfinex](https://www.bitfinex.com/)
 * [Huobi.pro](https://www.huobi.pro/)
 * [ZB](https://www.zb.com/)
 * [OKEx](https://www.okex.com/)
 * [Gate.io](https://gate.io/)
 * [Bittrex](https://bittrex.com/)
 * [HitBTC](https://hitbtc.com/)
 * ~~[BigONE](https://big.one/)~~
 * [Poloniex](https://poloniex.com/)
 * [Kraken](https://www.kraken.com/)
 * [Coinbase](https://www.coinbase.com/)
 * _still adding..._
 
### Installation

#### Homebrew

```bash
# WIP
```

#### `curl | bash` style downloads to `/usr/local/bin`
```bash
$ curl -sfL https://raw.githubusercontent.com/polyrabbit/my-token/master/install.sh | bash -s -- -d -b /usr/local/bin
```

#### Using [Go](https://golang.org/) (1.12+)
```bash
$ go get -u github.com/polyrabbit/my-token
```

#### Manually
Download from [release page](https://github.com/polyrabbit/my-token/releases/latest) and extract the tarbal into /usr/bin or your `PATH` directory.

### Usage

```
$ mt --help

Usage: mt [Options] [Exchange1.Token1 Exchange2.Token2.<api_key> ...]

Track token prices of your favorite exchanges in the terminal

Options:
  -v, --Version                            Show Version number
  -d, --debug                              Enable debug mode
  -l, --list-exchanges                     List supported exchanges
  -r, --refresh int                        Auto refresh on every specified seconds, note every exchange has a rate limit,
                                           too frequent refresh may cause your IP banned by their servers
  -c, --config-file string                 Config file path, use "--example-config-file <path>" to generate an example config file,
                                           by default my-token uses "my_token.yml" in current directory or $HOME as config file
      --example-config-file string[="-"]   Generate example config file to the specified file path, by default it outputs to stdout
  -s, --show strings                       Only show comma-separated columns (default [Symbol,Price,%Change(1h),%Change(24h),Source,Updated])
  -p, --proxy string                       Proxy used when sending HTTP request
                                           (eg. "http://localhost:7777", "https://localhost:7777", "socks5://localhost:1080")
  -t, --timeout int                        HTTP request timeout in seconds (default 20)

Space-separated exchange.token pairs:
  Specify which exchange and token pair to query, different exchanges use different forms to express tokens/trading pairs, refer to their URLs to find the format (eg. "Bitfinex.BTCUSDT"). Optionally you can set api_key in the third place.

Find help/updates from here - https://github.com/polyrabbit/my-token
```

* #### Display latest market prices for for `BNBUSDT`, `BTCUSDT` from `Binance` and `HTUSDT` from `Huobi`

```bash
$ mt binance.BNBUSDT binance.BTCUSDT Huobi.HTUSDT
```

Here `Binance` and `Huobi` can be replaced by any supported exchanges, and different exchanges use different forms to express tokens/symbols/markets, refer to their URLs to find the format.

* #### Auto-refresh on every 10 seconds

```bash
$ mt -r 10 binance.BNBUSDT binance.BTCUSDT Huobi.HTUSDT
```

NOTE: some exchanges has a strict rate limit, too frequent refresh may cause your IP banned by their servers.

* #### Show specified columns only

```bash
$ mt --show Symbol,Price binance.BTCUSDT
```

See issue [#3](https://github.com/polyrabbit/my-token/issues/3) for a discussion on this feature.

* #### Run with options from a configuration file

```bash
$ mt -c my_token.example.yaml
```

my-token can also read options from configuration file, see the attached [my_token.example.yaml](my_token.example.yaml) for its format. By default my-token searches configuration file `my_token.yml` in current directory and `$HOME`, so you can compose a `my_token.yml`, place it in your `$HOME` and just type `mt` to get all pre-defined prices. 

```bash
$ # Generate an example config file to my $HOME directory
$ mt --example-config-file=$HOME/my_token.yml
$
$
$ # my-token will search for configuration file "my_token.yml" in current directory and "$HOME" by default
$ mt       # <--- This is also the way I used most freqently 
```

### Thanks

 * Inspired by [coinmon](https://github.com/bichenkk/coinmon)

### License

The MIT License (MIT) - see [LICENSE.md](https://github.com/polyrabbit/my-token/blob/master/LICENSE) for more details
