package main

import (
	"bytes"
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
)

func renderPDF(ctx context.Context, cfg Config, workDir string) ([]string, error) {
	if cfg.DPI <= 0 {
		cfg.DPI = 144
	}
	prefix := filepath.Join(workDir, "page")
	args := []string{"-jpeg", "-r", strconv.Itoa(cfg.DPI), cfg.PDFPath, prefix}
	cmd := exec.CommandContext(ctx, "pdftoppm", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("pdftoppm failed: %w", err)
	}
	pattern := prefix + "-*.jpg"
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		return nil, errors.New("no rendered pages found")
	}
	sort.Slice(matches, func(i, j int) bool {
		return pageIndexFromName(matches[i]) < pageIndexFromName(matches[j])
	})
	return matches, nil
}

func countPDFPages(ctx context.Context, pdfPath string) (int, error) {
	cmd := exec.CommandContext(ctx, "pdfinfo", pdfPath)
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("pdfinfo failed: %w", err)
	}
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "Pages:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				if total, convErr := strconv.Atoi(parts[1]); convErr == nil {
					return total, nil
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, err
	}
	return 0, errors.New("failed to determine page count from pdfinfo output")
}

func renderSinglePage(ctx context.Context, cfg Config, workDir string, page int) (string, error) {
	if cfg.DPI <= 0 {
		cfg.DPI = 144
	}
	prefix := filepath.Join(workDir, "page")
	args := []string{
		"-jpeg",
		"-r", strconv.Itoa(cfg.DPI),
		"-f", strconv.Itoa(page),
		"-l", strconv.Itoa(page),
		cfg.PDFPath,
		prefix,
	}
	cmd := exec.CommandContext(ctx, "pdftoppm", args...)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("pdftoppm failed on page %d: %w", page, err)
	}
	return findRenderedImage(prefix, page)
}

func findRenderedImage(prefix string, page int) (string, error) {
	candidates := []string{
		fmt.Sprintf("%s-%d.jpg", prefix, page),
		fmt.Sprintf("%s-%02d.jpg", prefix, page),
		fmt.Sprintf("%s-%03d.jpg", prefix, page),
		fmt.Sprintf("%s-%04d.jpg", prefix, page),
		fmt.Sprintf("%s-%05d.jpg", prefix, page),
		fmt.Sprintf("%s-%06d.jpg", prefix, page),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		} else if err != nil && !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
	}
	matches, err := filepath.Glob(fmt.Sprintf("%s-*.jpg", prefix))
	if err != nil {
		return "", err
	}
	for _, match := range matches {
		if pageIndexFromName(match) == page-1 {
			return match, nil
		}
	}
	return "", fmt.Errorf("rendered image not found for page %d", page)
}

func renderPDFAsync(ctx context.Context, cfg Config, workDir string) (<-chan pageJob, <-chan error, int, error) {
	totalPages, err := countPDFPages(ctx, cfg.PDFPath)
	if err != nil {
		return nil, nil, 0, err
	}
	buffer := cfg.MaxConcurrency
	if buffer <= 0 {
		buffer = cfg.RenderWorkers
	}
	if buffer <= 0 {
		buffer = runtime.NumCPU()
	}
	jobs := make(chan pageJob, maxInt(buffer, 1))
	errChan := make(chan error, 1)
	go func() {
		defer close(jobs)
		defer close(errChan)
		renderCtx, cancel := context.WithCancel(ctx)
		defer cancel()
		workers := cfg.RenderWorkers
		if workers <= 0 {
			workers = maxInt(runtime.NumCPU(), 2)
			if workers > 8 {
				workers = 8
			}
		}
		sem := make(chan struct{}, workers)
		var wg sync.WaitGroup
		var once sync.Once
		sendErr := func(err error) {
			if err == nil {
				return
			}
			once.Do(func() {
				errChan <- err
				cancel()
			})
		}
		for page := 1; page <= totalPages; page++ {
			if renderCtx.Err() != nil {
				break
			}
			wg.Add(1)
			sem <- struct{}{}
			go func(p int) {
				defer wg.Done()
				defer func() { <-sem }()
				path, renderErr := renderSinglePage(renderCtx, cfg, workDir, p)
				if renderErr != nil {
					sendErr(renderErr)
					return
				}
				job := pageJob{index: p - 1, imagePath: path}
				select {
				case jobs <- job:
				case <-renderCtx.Done():
				}
			}(page)
		}
		wg.Wait()
	}()
	return jobs, errChan, totalPages, nil
}

func firstError(ch <-chan error) error {
	for err := range ch {
		if err != nil {
			return err
		}
	}
	return nil
}

func pageIndexFromName(path string) int {
	base := filepath.Base(path)
	idx := strings.LastIndex(base, "-")
	if idx >= 0 {
		number := strings.TrimSuffix(base[idx+1:], ".jpg")
		if v, err := strconv.Atoi(number); err == nil {
			return v - 1
		}
	}
	return 0
}
