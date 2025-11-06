package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	httpClientMu          sync.Mutex
	sharedHTTPClient      *http.Client
	sharedHTTPClientLimit int
)

func getHTTPClient(maxConns int) *http.Client {
	if maxConns <= 0 {
		maxConns = 2
	}
	httpClientMu.Lock()
	defer httpClientMu.Unlock()
	if sharedHTTPClient != nil && maxConns <= sharedHTTPClientLimit {
		return sharedHTTPClient
	}
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		MaxConnsPerHost:     maxInt(maxConns, 4),
		MaxIdleConnsPerHost: maxInt(maxConns, 4),
		MaxIdleConns:        maxInt(maxConns*2, 32),
		IdleConnTimeout:     90 * time.Second,
	}
	sharedHTTPClient = &http.Client{
		Transport: transport,
	}
	sharedHTTPClientLimit = maxConns
	return sharedHTTPClient
}

func runInference(ctx context.Context, cfg Config, imageB64 string) (string, error) {
	const maxAttempts = 3
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		text, err := invokeInference(ctx, cfg, imageB64)
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(text) != "" {
			return text, nil
		}
		if attempt < maxAttempts {
			time.Sleep(200 * time.Millisecond)
			continue
		}
		fmt.Fprintf(os.Stderr, "pdfworker warning: empty response text after %d attempts (task=%s)\n", attempt, cfg.TaskID)
		return "", nil
	}
	return "", nil
}

func invokeInference(ctx context.Context, cfg Config, imageB64 string) (string, error) {
	reqPayload := inferenceRequest{
		Prompt:    cfg.Prompt,
		ImageB64:  imageB64,
		BaseSize:  cfg.BaseSize,
		ImageSize: cfg.ImageSize,
		CropMode:  cfg.CropMode,
	}
	body, err := json.Marshal(reqPayload)
	if err != nil {
		return "", err
	}
	client := getHTTPClient(cfg.MaxConcurrency)
	reqCtx := ctx
	var cancel context.CancelFunc
	if cfg.RequestTimeout > 0 {
		reqCtx, cancel = context.WithTimeout(ctx, time.Duration(cfg.RequestTimeout)*time.Second)
		defer cancel()
	}
	request, err := http.NewRequestWithContext(reqCtx, http.MethodPost, cfg.InferURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/json")
	if cfg.AuthToken != "" {
		request.Header.Set("X-Internal-Token", cfg.AuthToken)
	}
	resp, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return "", fmt.Errorf("inference failed: status %d: %s", resp.StatusCode, string(data))
	}
	var parsed inferenceResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", err
	}
	return parsed.Text, nil
}
