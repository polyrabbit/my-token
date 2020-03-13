package http

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/spf13/viper"

	_ "github.com/polyrabbit/my-token/config" // config should be initialized first
	"github.com/sirupsen/logrus"
)

// Thread save
var HTTPClient *http.Client

func init() {
	timeout := viper.GetInt("timeout")
	logrus.Debugf("HTTP request timeout is set to %d seconds", timeout)
	HTTPClient = &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}

	rawProxyURL := viper.GetString("proxy")
	if rawProxyURL != "" {
		proxyURL, err := url.Parse(rawProxyURL)
		if err != nil {
			logrus.Warnf("Failed to parse proxy URL: %s, error: %v, using system proxy", rawProxyURL, err)
		} else {
			transport := &http.Transport{
				Proxy: http.ProxyURL(proxyURL),
			}
			logrus.Debugf("Using proxy %s", rawProxyURL)
			HTTPClient.Transport = transport
		}
	}
}

type HTTPError struct {
	Status string
	Body   []byte
}

func (e *HTTPError) Error() string {
	return e.Status
}

func Get(rawURL string, params map[string]string) ([]byte, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		logrus.Fatalln(err)
	}
	if params != nil {
		query := url.Values{}
		for k, v := range params {
			query.Set(k, v)
		}
		parsedURL.RawQuery = query.Encode()
	}

	req, err := http.NewRequest("GET", parsedURL.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; my-token; +https://github.com/polyrabbit/my-token)")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Add("Cache-Control", "no-store")
	req.Header.Add("Cache-Control", "must-revalidate")

	resp, err := HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
		return nil, &HTTPError{resp.Status, respBytes}
	}
	return respBytes, err
}
