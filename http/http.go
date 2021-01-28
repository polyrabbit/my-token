package http

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Client struct {
	StdClient *http.Client
}

func New() *Client {
	// Thread safe
	stdClient := http.DefaultClient
	timeout := viper.GetInt("timeout")
	if timeout != 0 {
		logrus.Debugf("HTTP request timeout is set to %d seconds", timeout)
		stdClient = &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		}
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
			stdClient.Transport = transport
		}
	}
	return &Client{stdClient}
}

func (c *Client) Get(rawURL string, params map[string]string) ([]byte, error) {
	if params != nil {
		parsedURL, err := url.Parse(rawURL)
		if err != nil {
			return nil, fmt.Errorf("parse url %s: %w", rawURL, err)
		}
		query := url.Values{}
		for k, v := range params {
			query.Set(k, v)
		}
		parsedURL.RawQuery = query.Encode()
		rawURL = parsedURL.String()
	}

	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; my-token; +https://github.com/polyrabbit/my-token)")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Add("Cache-Control", "no-store")
	req.Header.Add("Cache-Control", "must-revalidate")

	resp, err := c.StdClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
		// Most non-200 responses have valid json body
		return respBytes, &ResponseError{resp.Status, respBytes}
	}
	return respBytes, err
}

type ResponseError struct {
	Status string
	Body   []byte
}

func (e *ResponseError) Error() string {
	return "HTTP " + e.Status + ", body " + string(e.Body[:200])
}
