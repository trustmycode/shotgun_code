package provider

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

const maxProviderResponseBytes = 8 * 1024 * 1024

var providerHTTPClient = &http.Client{
	Timeout:   2 * time.Minute,
	Transport: responseLimitTransport{base: http.DefaultTransport},
}

type responseLimitTransport struct {
	base http.RoundTripper
}

func (transport responseLimitTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	response, err := transport.base.RoundTrip(request)
	if err != nil {
		return nil, err
	}
	response.Body = struct {
		io.Reader
		io.Closer
	}{
		Reader: io.LimitReader(response.Body, maxProviderResponseBytes+1),
		Closer: response.Body,
	}
	return response, nil
}

func decodeLimitedJSON(body io.Reader, target any) error {
	data, err := io.ReadAll(io.LimitReader(body, maxProviderResponseBytes+1))
	if err != nil {
		return fmt.Errorf("read provider response: %w", err)
	}
	if len(data) > maxProviderResponseBytes {
		return errors.New("provider response exceeds the 8 MiB limit")
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("decode provider response: %w", err)
	}
	return nil
}
