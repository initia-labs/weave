package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	maxRetries = 3
	baseDelay  = 1 * time.Second

	downloadBufferSize = 65536
)

// HTTPClient defines the logic for making HTTP requests.
type HTTPClient struct{}

// NewHTTPClient creates and returns a new HTTPClient instance.
func NewHTTPClient() *HTTPClient {
	return &HTTPClient{}
}

// Get performs an HTTP GET request.
// It can either unmarshal a JSON response into the provided result or return the raw response data directly.
func (c *HTTPClient) Get(baseURL, additionalPath string, params map[string]string, result interface{}) ([]byte, error) {
	fullURL := constructURL(baseURL, additionalPath, params)

	body, err := c.getWithRetry(fullURL)
	if err != nil {
		return nil, err
	}

	if result != nil {
		if err := json.Unmarshal(body, result); err != nil {
			return nil, fmt.Errorf("failed to parse JSON response: %w", err)
		}
	}

	return body, nil
}

// getWithRetry performs the HTTP GET request with retry logic, including exponential backoff.
func (c *HTTPClient) getWithRetry(endpoint string) ([]byte, error) {
	var lastErr error

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// if baseURL is api.github.com
	if strings.HasPrefix(endpoint, "https://api.github.com") && os.Getenv("GITHUB_TOKEN") != "" {
		// add GITHUB_TOKEN to headers
		req.Header.Add("Authorization", "Bearer "+os.Getenv("GITHUB_TOKEN"))
	}

	for attempt := 1; attempt <= maxRetries; attempt++ {
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("attempt %d: request error: %w", attempt, err)
		} else {
			if resp.StatusCode != http.StatusOK {
				lastErr = fmt.Errorf("attempt %d: unexpected status code %d", attempt, resp.StatusCode)
			} else {
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					lastErr = fmt.Errorf("attempt %d: failed to read response body: %w", attempt, err)
				} else {
					return body, nil
				}
			}
			err = resp.Body.Close()
			if err != nil {
				return nil, err
			}
		}
		time.Sleep(baseDelay * time.Duration(attempt))
	}

	return nil, fmt.Errorf("all %d attempts failed: %w", maxRetries, lastErr)
}

// Post performs an HTTP POST request.
// It can either unmarshal a JSON response into the provided result or return the raw response data directly.
func (c *HTTPClient) Post(baseURL, additionalPath string, headers map[string]string, body []byte, result interface{}) ([]byte, error) {
	fullURL := constructURL(baseURL, additionalPath, map[string]string{})

	response, err := c.postWithRetry(fullURL, headers, body)
	if err != nil {
		return nil, err
	}

	if result != nil {
		if err := json.Unmarshal(response, result); err != nil {
			return nil, fmt.Errorf("failed to parse JSON response: %w", err)
		}
	}

	return response, nil
}

// postWithRetry performs the HTTP POST request with retry logic, including exponential backoff.
func (c *HTTPClient) postWithRetry(endpoint string, headers map[string]string, body []byte) ([]byte, error) {
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(body))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		for key, value := range headers {
			req.Header.Set(key, value)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("attempt %d: request error: %w", attempt, err)
		} else {
			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				lastErr = fmt.Errorf("attempt %d: unexpected status code %d", attempt, resp.StatusCode)
			} else {
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					lastErr = fmt.Errorf("attempt %d: failed to read response body: %w", attempt, err)
				} else {
					return body, nil
				}
			}
			err = resp.Body.Close()
			if err != nil {
				return nil, fmt.Errorf("failed to close response body: %w", err)
			}
		}

		time.Sleep(baseDelay * time.Duration(attempt))
	}

	return nil, fmt.Errorf("all %d attempts failed: %w", maxRetries, lastErr)
}

// DownloadFile downloads a file from the specified URL
// and updates the current progress using the provided progress pointer.
func (c *HTTPClient) DownloadFile(url string, dest string, progress, totalSize *int64) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to connect to URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download: received status code %d", resp.StatusCode)
	}

	if totalSize != nil {
		*totalSize = resp.ContentLength
		if *totalSize <= 0 {
			*totalSize = 1
		}
	}

	file, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer file.Close()

	buffer := make([]byte, downloadBufferSize)
	var totalDownloaded int64
	for {
		n, err := resp.Body.Read(buffer)
		if err != nil && err != io.EOF {
			return fmt.Errorf("error during file download: %w", err)
		}
		if n == 0 {
			break
		}

		if _, err := file.Write(buffer[:n]); err != nil {
			return fmt.Errorf("failed to write to file: %w", err)
		}

		totalDownloaded += int64(n)
		if progress != nil {
			*progress = totalDownloaded
		}
	}

	return nil
}

// DownloadAndValidateFile does the HTTPClient.DownloadFile but with additional validation
func (c *HTTPClient) DownloadAndValidateFile(url string, dest string, progress, totalSize *int64, validateFn func(string) error) error {
	if err := c.DownloadFile(url, dest, progress, totalSize); err != nil {
		return err
	}

	if err := validateFn(dest); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	return nil
}

// constructURL builds a complete URL with optional query parameters.
func constructURL(baseURL, additionalPath string, params map[string]string) string {
	u, _ := url.Parse(baseURL)
	if additionalPath != "" {
		u.Path = strings.TrimRight(u.Path, "/") + "/" + strings.TrimLeft(additionalPath, "/")
	} else {
		u.Path = strings.TrimRight(u.Path, "/")
	}

	if len(params) > 0 {
		query := u.Query()
		for key, value := range params {
			query.Add(key, value)
		}
		u.RawQuery = query.Encode()
	}
	return u.String()
}
