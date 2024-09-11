package services

import (
	"time"

	"github.com/gojek/heimdall/v7"
	"github.com/gojek/heimdall/v7/httpclient"
)

type ServiceHTTP struct{}

func (service *ServiceHTTP) httpClient(retries int) *httpclient.Client {
	client := httpclient.NewClient(
		httpclient.WithHTTPTimeout(10 * time.Second),
	)

	if retries > 0 {
		backoffInterval := 200 * time.Millisecond
		maximumJitterInterval := 1 * time.Second
		backoff := heimdall.NewConstantBackoff(backoffInterval, maximumJitterInterval)
		retrier := heimdall.NewRetrier(backoff)

		httpclient.WithRetrier(retrier)(client)
		httpclient.WithRetryCount(retries)(client)
	}

	return client
}
