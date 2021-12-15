package http

import (
    "io/ioutil"
    "net/http"
    "net/url"
    "time"

    "github.com/polyrabbit/my-token/config"
    "github.com/sirupsen/logrus"
)

type Client struct {
    StdClient *http.Client
}

func New(cfg *config.Config) *Client {
    // Thread safe
    stdClient := http.DefaultClient
    timeout := cfg.Timeout
    if timeout != 0 {
        logrus.Debugf("HTTP request timeout is set to %d seconds", timeout)
        stdClient = &http.Client{
            Timeout: time.Duration(timeout) * time.Second,
        }
    }

    rawProxyURL := cfg.Proxy
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

func (c *Client) Get(rawURL string, opts ...RequestOption) ([]byte, error) {
    option := defaultRequestOptions
    for _, o := range opts {
        o(&option)
    }

    rawURL = option.AppendQuery(rawURL)
    req, err := http.NewRequest("GET", rawURL, nil)
    if err != nil {
        return nil, err
    }
    option.SetHeader(req.Header)

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

type RequestOption func(*RequestOptions)

type RequestOptions struct {
    query  map[string]string
    header map[string]string
}

func WithQuery(query map[string]string) RequestOption {
    return func(o *RequestOptions) {
        o.query = query
    }
}

func WithHeader(header map[string]string) RequestOption {
    return func(o *RequestOptions) {
        o.header = header
    }
}

var defaultRequestOptions = RequestOptions{
    header: map[string]string{
        "Accept":        "application/json",
        "User-Agent":    "Mozilla/5.0 (compatible; my-token; +https://github.com/polyrabbit/my-token)",
        "Cache-Control": "no-store, no-cache, private",
    },
}

func (o *RequestOptions) AppendQuery(rawURL string) string {
    if o.query == nil {
        return rawURL
    }
    parsedURL, err := url.Parse(rawURL)
    if err != nil {
        return rawURL
    }
    query := url.Values{}
    for k, v := range o.query {
        query.Set(k, v)
    }
    parsedURL.RawQuery = query.Encode()
    return parsedURL.String()
}

func (o *RequestOptions) SetHeader(req http.Header) {
    for k, v := range o.header {
        req.Set(k, v)
    }
}
