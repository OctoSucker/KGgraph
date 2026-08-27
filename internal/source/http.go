package source

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const userAgent = "kggraph-source/1.0 (research corpus collector; contact: user)"

var httpClient = &http.Client{Timeout: 60 * time.Second}

func httpGet(ctx context.Context, url string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt < 6; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", userAgent)
		req.Header.Set("Accept", "text/html,application/json,*/*")
		resp, err := httpClient.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(3*(attempt+1)) * time.Second)
			continue
		}
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			time.Sleep(time.Duration(3*(attempt+1)) * time.Second)
			continue
		}
		if resp.StatusCode == http.StatusTooManyRequests {
			retry := 20
			if v := resp.Header.Get("Retry-After"); v != "" {
				if n, err := strconv.Atoi(v); err == nil {
					retry = n + 2
				}
			}
			lastErr = fmt.Errorf("rate limited (attempt %d)", attempt+1)
			time.Sleep(time.Duration(retry) * time.Second)
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			lastErr = fmt.Errorf("http %d from %s", resp.StatusCode, url)
			time.Sleep(time.Duration(3*(attempt+1)) * time.Second)
			continue
		}
		return body, nil
	}
	return nil, lastErr
}

func httpGetString(ctx context.Context, url string) (string, error) {
	body, err := httpGet(ctx, url)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(body)), nil
}
