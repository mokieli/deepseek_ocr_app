package main

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "", "path to config json")
	flag.Parse()
	writer := newEventWriter(os.Stdout)
	defer writer.flush()

	if configPath == "" {
		writer.Error(errors.New("missing --config"))
		os.Exit(1)
	}
	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		writer.Error(fmt.Errorf("failed to read config: %w", err))
		os.Exit(1)
	}
	var cfg Config
	if err := json.Unmarshal(configBytes, &cfg); err != nil {
		writer.Error(fmt.Errorf("invalid config: %w", err))
		os.Exit(1)
	}
	ensureDefaultConfig(&cfg)
	if cfg.PDFPath == "" || cfg.OutputDir == "" || cfg.InferURL == "" || cfg.Prompt == "" {
		writer.Error(errors.New("config missing required fields"))
		os.Exit(1)
	}
	if err := os.MkdirAll(cfg.OutputDir, 0o755); err != nil {
		writer.Error(err)
		os.Exit(1)
	}
	imagesDir := filepath.Join(cfg.OutputDir, "images")
	if err := os.MkdirAll(imagesDir, 0o755); err != nil {
		writer.Error(err)
		os.Exit(1)
	}
	workDir := filepath.Join(cfg.OutputDir, ".worker")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		writer.Error(err)
		os.Exit(1)
	}
	defer os.RemoveAll(workDir)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var progressTotal int64 = 1
	var pagesCompleted int64
	var pagesQueued int64
	var pagesTotal int64
	reportProgress := func(current int, message string) {
		total := int(atomic.LoadInt64(&progressTotal))
		if total < 1 {
			total = 1
		}
		pc := int(atomic.LoadInt64(&pagesCompleted))
		pt := int(atomic.LoadInt64(&pagesTotal))
		payload := progressPayload{
			Current: current,
			Total:   total,
			Message: message,
		}
		if pt > 0 {
			payload.PagesTotal = &pt
			payload.PagesCompleted = &pc
		}
		writer.Progress(payload)
	}

	emitEmptyResult := func() {
		reportProgress(0, "PDF 中未检测到页面")
		writer.Result(map[string]interface{}{
			"markdown_file": "",
			"raw_json_file": "",
			"pages":         []interface{}{},
			"images":        []interface{}{},
			"progress": map[string]interface{}{
				"current": 0,
				"total":   0,
				"percent": 100.0,
				"message": "已完成",
				"pages_completed": 0,
				"pages_total":     0,
			},
		})
	}

	reportProgress(0, "正在准备 PDF 渲染")
	jobStream, renderErrStream, totalPages, err := renderPDFAsync(ctx, cfg, workDir)
	if err != nil {
		reportProgress(0, "PDF 渲染回退到串行模式")
		pages, fallbackErr := renderPDF(ctx, cfg, workDir)
		if fallbackErr != nil {
			writer.Error(fallbackErr)
			os.Exit(1)
		}
		totalPages = len(pages)
		if totalPages == 0 {
			emitEmptyResult()
			return
		}
		fallbackJobs := make(chan pageJob, maxInt(totalPages, 1))
		fallbackErrStream := make(chan error, 1)
		go func() {
			defer close(fallbackJobs)
			defer close(fallbackErrStream)
			for i, path := range pages {
				fallbackJobs <- pageJob{index: i, imagePath: path}
			}
		}()
		jobStream = fallbackJobs
		renderErrStream = fallbackErrStream
	} else {
		if totalPages == 0 {
			emitEmptyResult()
			return
		}
	}

	atomic.StoreInt64(&pagesTotal, int64(totalPages))

	initialTotal := maxInt(totalPages+3, 1)
	atomic.StoreInt64(&progressTotal, int64(initialTotal))
	reportProgress(0, "PDF 页面渲染中")

	results := make([]pageResult, totalPages)
	errChan := make(chan error, 1)
	inflight := int64(0)
	completed := int64(0)

	semCapacity := cfg.MaxConcurrency
	if semCapacity <= 0 {
		semCapacity = 1
	}
	sem := make(chan struct{}, semCapacity)
	var wg sync.WaitGroup
	for job := range jobStream {
		wg.Add(1)
		sem <- struct{}{}
		queued := int(atomic.AddInt64(&pagesQueued, 1))
		reportProgress(int(atomic.LoadInt64(&completed)), fmt.Sprintf("已排队 %d/%d 页", queued, totalPages))
		go func(j pageJob) {
			defer wg.Done()
			defer func() { <-sem }()
			atomic.AddInt64(&inflight, 1)
			defer atomic.AddInt64(&inflight, -1)
			pageRes, err := processPage(ctx, cfg, j.index, j.imagePath, imagesDir)
			if err != nil {
				select {
				case errChan <- err:
				default:
				}
				return
			}
			results[j.index] = pageRes
			done := int(atomic.AddInt64(&completed, 1))
			atomic.StoreInt64(&pagesCompleted, int64(done))
			reportProgress(done, fmt.Sprintf("已完成 %d/%d 页", done, totalPages))
		}(job)
	}

	renderErr := firstError(renderErrStream)

	go func() {
		wg.Wait()
		close(errChan)
	}()

	if err := <-errChan; err != nil {
		writer.Error(err)
		os.Exit(1)
	}
	if renderErr != nil {
		writer.Error(renderErr)
		os.Exit(1)
	}

	pagesDone := int(atomic.LoadInt64(&completed))
	atomic.StoreInt64(&pagesCompleted, int64(pagesDone))
	reportProgress(pagesDone, "正在生成 Markdown 摘要")
	markdownPath := filepath.Join(cfg.OutputDir, "result.md")
	if err := writeMarkdown(markdownPath, results); err != nil {
		writer.Error(err)
		os.Exit(1)
	}

	pagesDone++
	reportProgress(pagesDone, "正在生成原始 JSON")
	rawJSONPath := filepath.Join(cfg.OutputDir, "raw.json")
	if err := writeJSON(rawJSONPath, results); err != nil {
		writer.Error(err)
		os.Exit(1)
	}
	pagesDone++

	var allAssets []string
	for _, page := range results {
		allAssets = append(allAssets, page.ImageAssets...)
	}

	entries := make([]archiveEntry, 0, len(allAssets)+2)
	entries = append(entries, archiveEntry{
		Name:   filepath.Base(markdownPath),
		Path:   markdownPath,
		Method: zip.Deflate,
	})
	entries = append(entries, archiveEntry{
		Name:   filepath.Base(rawJSONPath),
		Path:   rawJSONPath,
		Method: zip.Deflate,
	})
	for _, asset := range allAssets {
		fullPath := filepath.Join(cfg.OutputDir, asset)
		method := zip.Deflate
		if shouldStore(asset) {
			method = zip.Store
		}
		entries = append(entries, archiveEntry{
			Name:   asset,
			Path:   fullPath,
			Method: method,
		})
	}

	archiveTotal := len(entries)
	finalTotal := pagesDone + archiveTotal
	if finalTotal < pagesDone+1 {
		finalTotal = pagesDone + 1
	}
	atomic.StoreInt64(&progressTotal, int64(finalTotal))
	reportProgress(pagesDone, fmt.Sprintf("正在压缩结果资源 (0/%d)", archiveTotal))

	var archiveName string
	if cfg.OriginalFilename != "" {
	  archiveName = fmt.Sprintf("%s_PDF_OCR_Result.zip", cfg.OriginalFilename)
	} else {
	  archiveName = "result.zip"
	}
	archivePath := filepath.Join(cfg.OutputDir, archiveName)
	baseProgress := pagesDone
	if err := writeArchive(archivePath, entries, func(done int, total int, name string) {
		reportProgress(baseProgress+done, fmt.Sprintf("正在压缩结果资源 (%d/%d)", done, total))
	}); err != nil {
		writer.Error(err)
		os.Exit(1)
	}

	atomic.StoreInt64(&pagesCompleted, int64(totalPages))
	reportProgress(finalTotal, "全部页面处理完成")

	payload := map[string]interface{}{
		"markdown_file": filepath.Base(markdownPath),
		"raw_json_file": filepath.Base(rawJSONPath),
		"pages": func() []map[string]interface{} {
			out := make([]map[string]interface{}, len(results))
			for i, page := range results {
				out[i] = map[string]interface{}{
					"index":        page.Index,
					"page_number":  page.Index + 1,
					"markdown":     page.Markdown,
					"raw_text":     page.RawText,
					"image_assets": page.ImageAssets,
					"boxes":        page.Boxes,
				}
			}
			return out
		}(),
		"images":       allAssets,
		"archive_file": filepath.Base(archivePath),
		"progress": map[string]interface{}{
			"current": finalTotal,
			"total":   finalTotal,
			"percent": 100.0,
			"message": "已完成",
			"pages_completed": totalPages,
			"pages_total":     totalPages,
		},
	}
	payload["total_pages"] = totalPages
	writer.Result(payload)
}
