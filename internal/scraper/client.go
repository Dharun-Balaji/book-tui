package scraper

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

const defaultUserAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"

type Client struct {
	httpClient *http.Client
	mu         sync.Mutex
	next       map[string]time.Time
	Debugf     func(string, ...any)
}

func NewClient() *Client {
	return &Client{httpClient: &http.Client{Timeout: 30 * time.Second}, next: make(map[string]time.Time)}
}

func (client *Client) Fetch(ctx context.Context, sourceID string, rateLimit int, referer, url string) (string, error) {
	if rateLimit <= 0 {
		rateLimit = 60
	}
	for attempt := 0; attempt < 5; attempt++ {
		if err := client.wait(ctx, sourceID, rateLimit); err != nil {
			return "", err
		}
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return "", err
		}
		request.Header.Set("User-Agent", defaultUserAgent)
		request.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
		request.Header.Set("Accept-Language", "en-US,en;q=0.9")
		request.Header.Set("Referer", referer)
		request.Header.Set("Sec-Fetch-Dest", "document")
		request.Header.Set("Sec-Fetch-Mode", "navigate")
		request.Header.Set("Sec-Fetch-Site", "same-origin")
		response, err := client.httpClient.Do(request)
		if err != nil {
			return "", err
		}
		body, readErr := io.ReadAll(io.LimitReader(response.Body, 10<<20))
		response.Body.Close()
		if readErr != nil {
			return "", readErr
		}
		if response.StatusCode >= 200 && response.StatusCode < 300 {
			return string(body), nil
		}
		if response.StatusCode != http.StatusTooManyRequests && response.StatusCode != http.StatusServiceUnavailable {
			return "", fmt.Errorf("GET %s: %s", url, response.Status)
		}
		if attempt == 4 {
			return "", fmt.Errorf("GET %s: %s after retries", url, response.Status)
		}
		delay := time.Second << attempt
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(delay):
		}
	}
	return "", fmt.Errorf("unreachable")
}

func (client *Client) wait(ctx context.Context, sourceID string, rateLimit int) error {
	interval := time.Minute / time.Duration(rateLimit)
	client.mu.Lock()
	readyAt := client.next[sourceID]
	now := time.Now()
	if readyAt.Before(now) {
		readyAt = now
	}
	client.next[sourceID] = readyAt.Add(interval)
	client.mu.Unlock()
	if delay := time.Until(readyAt); delay > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}
	return nil
}
