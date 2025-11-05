package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"io"
	"hash/crc32"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Config struct {
	TaskID         string `json:"task_id"`
	PDFPath        string `json:"pdf_path"`
	OutputDir      string `json:"output_dir"`
	DPI            int    `json:"dpi"`
	Prompt         string `json:"prompt"`
	InferURL       string `json:"infer_url"`
	AuthToken      string `json:"auth_token"`
	BaseSize       int    `json:"base_size"`
	ImageSize      int    `json:"image_size"`
	CropMode       bool   `json:"crop_mode"`
	MaxConcurrency int    `json:"max_concurrency"`
	RenderWorkers  int    `json:"render_workers"`
	RequestTimeout int    `json:"request_timeout_seconds"`
}

type outputEvent struct {
	Type     string                 `json:"type"`
	Progress *progressPayload       `json:"progress,omitempty"`
	Payload  map[string]interface{} `json:"payload,omitempty"`
	Error    string                 `json:"error,omitempty"`
}

type progressPayload struct {
	Current int     `json:"current"`
	Total   int     `json:"total"`
	Percent float64 `json:"percent"`
	Message string  `json:"message"`
}

type inferenceRequest struct {
	Prompt    string `json:"prompt"`
	ImageB64  string `json:"image_base64"`
	BaseSize  int    `json:"base_size"`
	ImageSize int    `json:"image_size"`
	CropMode  bool   `json:"crop_mode"`
}

type inferenceResponse struct {
	Text string `json:"text"`
}

type pageResult struct {
	Index       int
	Markdown    string
	RawText     string
	ImageAssets []string
	Boxes       []map[string]interface{}
}

type pageJob struct {
	index     int
	imagePath string
}

type archiveEntry struct {
	Name   string
	Path   string
	Method uint16
}

type eventWriter struct {
	enc *json.Encoder
	w   *bufio.Writer
	mu  sync.Mutex
}

var (
	httpClientMu          sync.Mutex
	sharedHTTPClient      *http.Client
	sharedHTTPClientLimit int
)

func newEventWriter(writer io.Writer) *eventWriter {
	buf := bufio.NewWriter(writer)
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	return &eventWriter{enc: enc, w: buf}
}

func (e *eventWriter) flush() {
	_ = e.w.Flush()
}

func (e *eventWriter) Progress(current, total int, message string) {
	percent := 0.0
	if total > 0 {
		percent = float64(current) / float64(total) * 100.0
		if percent > 100.0 {
			percent = 100.0
		}
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	_ = e.enc.Encode(outputEvent{
		Type: "progress",
		Progress: &progressPayload{
			Current: current,
			Total:   total,
			Percent: percent,
			Message: message,
		},
	})
	e.flush()
}

func (e *eventWriter) Result(payload map[string]interface{}) {
	e.mu.Lock()
	defer e.mu.Unlock()
	_ = e.enc.Encode(outputEvent{
		Type:    "result",
		Payload: payload,
	})
	e.flush()
}

func (e *eventWriter) Error(err error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	_ = e.enc.Encode(outputEvent{
		Type:  "error",
		Error: err.Error(),
	})
	e.flush()
}

var detectionBlock = regexp.MustCompile(`(?s)<\|ref\|>(?P<label>.*?)<\|/ref\|>\s*<\|det\|>(?P<coords>.*?)<\|/det\|>`)

var textualLabelKeywords = []string{
	"text",
	"title",
	"subtitle",
	"sub_title",
	"caption",
	"paragraph",
	"header",
	"footer",
	"footnote",
	"list",
	"figure",
	"table",
	"page_number",
}

var fullWidthMap = strings.NewReplacer(
	"，", ",",
	"。", ".",
	"；", ",",
	"：", ":",
	"【", "[",
	"】", "]",
	"（", "(",
	"）", ")",
	"、", ",",
	"％", "%",
	"－", "-",
)

func isTextualLabel(label string) bool {
	normalized := strings.TrimSpace(strings.ToLower(label))
	if normalized == "" {
		return true
	}
	for _, keyword := range textualLabelKeywords {
		if strings.Contains(normalized, keyword) {
			return true
		}
	}
	return false
}

func sanitizeCoordsText(raw string) string {
	cleaned := fullWidthMap.Replace(raw)
	cleaned = regexp.MustCompile(`<\|.*?\|>`).ReplaceAllString(cleaned, "")
	cleaned = strings.TrimSpace(cleaned)
	start := strings.Index(cleaned, "[")
	end := strings.LastIndex(cleaned, "]")
	if start >= 0 && end >= start {
		return cleaned[start : end+1]
	}
	return ""
}

func parseCoords(raw string) [][]float64 {
	cleaned := sanitizeCoordsText(raw)
	if cleaned == "" {
		return nil
	}
	var data any
	if err := json.Unmarshal([]byte(cleaned), &data); err != nil {
		return nil
	}
	switch v := data.(type) {
	case []any:
		return normalizeCoordsSlice(v)
	default:
		return nil
	}
}

func normalizeCoordsSlice(items []any) [][]float64 {
	if len(items) == 4 {
		allNumeric := true
		box := make([]float64, 4)
		for i := 0; i < 4; i++ {
			val, ok := asFloat(items[i])
			if !ok {
				allNumeric = false
				break
			}
			box[i] = val
		}
		if allNumeric {
			return [][]float64{box}
		}
	}
	var boxes [][]float64
	for _, item := range items {
		switch t := item.(type) {
		case []any:
			if len(t) >= 4 && isNumericSlice(t[:4]) {
				box := make([]float64, 4)
				for i := 0; i < 4; i++ {
					box[i], _ = asFloat(t[i])
				}
				boxes = append(boxes, box)
				continue
			}
			if len(t) >= 2 {
				first, ok1 := t[0].([]any)
				second, ok2 := t[1].([]any)
				if ok1 && ok2 && len(first) >= 2 && len(second) >= 2 {
					x1, okx1 := asFloat(first[0])
					y1, oky1 := asFloat(first[1])
					x2, okx2 := asFloat(second[0])
					y2, oky2 := asFloat(second[1])
					if okx1 && oky1 && okx2 && oky2 {
						boxes = append(boxes, []float64{x1, y1, x2, y2})
					}
				}
			}
		}
	}
	return boxes
}

func asFloat(value any) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case json.Number:
		f, err := v.Float64()
		if err != nil {
			return 0, false
		}
		return f, true
	default:
		return 0, false
	}
}

func isNumericSlice(values []any) bool {
	for _, v := range values {
		if _, ok := asFloat(v); !ok {
			return false
		}
	}
	return true
}

func scaleBox(box []float64, width, height int) [4]int {
	if len(box) < 4 {
		return [4]int{}
	}
	return [4]int{
		int(box[0] / 999.0 * float64(width)),
		int(box[1] / 999.0 * float64(height)),
		int(box[2] / 999.0 * float64(width)),
		int(box[3] / 999.0 * float64(height)),
	}
}

func parseDetections(raw string, width, height int) []map[string]interface{} {
	var boxes []map[string]interface{}
	matches := detectionBlock.FindAllStringSubmatch(raw, -1)
	if matches == nil {
		return boxes
	}
	labelIdx := detectionBlock.SubexpIndex("label")
	coordsIdx := detectionBlock.SubexpIndex("coords")
	for _, match := range matches {
		if labelIdx < 0 || coordsIdx < 0 || len(match) <= coordsIdx {
			continue
		}
		label := strings.TrimSpace(match[labelIdx])
		coords := parseCoords(match[coordsIdx])
		for _, box := range coords {
			scaled := scaleBox(box, width, height)
			if scaled[2] <= scaled[0] || scaled[3] <= scaled[1] {
				continue
			}
			boxes = append(boxes, map[string]interface{}{
				"label": label,
				"box":   []int{scaled[0], scaled[1], scaled[2], scaled[3]},
			})
		}
	}
	return boxes
}

func replaceDetectionBlocks(raw string, img image.Image, pageIndex int, imagesDir string) (string, []string, error) {
	labelIdx := detectionBlock.SubexpIndex("label")
	coordsIdx := detectionBlock.SubexpIndex("coords")
	if labelIdx < 0 || coordsIdx < 0 {
		return raw, nil, errors.New("invalid detection regex configuration")
	}
	var mu sync.Mutex
	var assets []string
	width := img.Bounds().Dx()
	height := img.Bounds().Dy()
	builder := strings.Builder{}
	cursor := 0
	locs := detectionBlock.FindAllStringSubmatchIndex(raw, -1)
	for _, loc := range locs {
		start := loc[0]
		end := loc[1]
		builder.WriteString(raw[cursor:start])
		cursor = end
		subs := detectionBlock.FindStringSubmatch(raw[start:end])
		label := strings.TrimSpace(subs[labelIdx])
		coords := subs[coordsIdx]
		if strings.EqualFold(label, "image") {
			boxes := parseCoords(coords)
			var blocks []string
			for _, box := range boxes {
				scaled := scaleBox(box, width, height)
				if scaled[2] <= scaled[0] || scaled[3] <= scaled[1] {
					continue
				}
				name := fmt.Sprintf("images/page-%d-img-%d.jpg", pageIndex, len(blocks))
				fullPath := filepath.Join(imagesDir, filepath.Base(name))
				if err := cropAndSave(img, scaled, fullPath); err != nil {
					return "", nil, err
				}
				mu.Lock()
				assets = append(assets, name)
				mu.Unlock()
				blocks = append(blocks, fmt.Sprintf("![](%s)", name))
			}
			if len(blocks) > 0 {
				builder.WriteString(strings.Join(blocks, "\n"))
			}
		} else if isTextualLabel(label) {
			// drop textual labels entirely
		} else {
			builder.WriteString(fmt.Sprintf("<!-- %s -->", label))
		}
	}
	builder.WriteString(raw[cursor:])
	processed := builder.String()
	processed = strings.ReplaceAll(processed, "<|grounding|>", "")
	processed = regexp.MustCompile(`\n{3,}`).ReplaceAllString(processed, "\n\n")
	processed = strings.TrimSpace(processed)
	return processed, assets, nil
}

func cropAndSave(img image.Image, box [4]int, dest string) error {
	rect := image.Rect(box[0], box[1], box[2], box[3])
	if !rect.In(img.Bounds()) {
		rect = rect.Intersect(img.Bounds())
	}
	if rect.Empty() {
		return nil
	}
	cropped := image.NewRGBA(image.Rect(0, 0, rect.Dx(), rect.Dy()))
	draw.Draw(cropped, cropped.Bounds(), img, rect.Min, draw.Src)
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	file, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer file.Close()
	opts := jpeg.Options{Quality: 95}
	return jpeg.Encode(file, cropped, &opts)
}

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
	// Ensure deterministic order by sorting by page number
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

func encodeImageToBase64(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	buf := make([]byte, base64.StdEncoding.EncodedLen(len(data)))
	base64.StdEncoding.Encode(buf, data)
	return string(buf), nil
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

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
	if parsed.Text == "" {
		return "", errors.New("empty response text")
	}
	return parsed.Text, nil
}

func loadImage(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}
	return img, nil
}

func writeMarkdown(outputPath string, pages []pageResult) error {
	var blocks []string
	for _, page := range pages {
		header := fmt.Sprintf("<!-- page:%d -->", page.Index+1)
		content := strings.TrimSpace(page.Markdown)
		if content == "" {
			blocks = append(blocks, header)
			continue
		}
		blocks = append(blocks, fmt.Sprintf("%s\n\n%s", header, content))
	}
	joined := strings.Join(blocks, "\n\n---\n\n")
	return os.WriteFile(outputPath, []byte(joined), 0o644)
}

func writeJSON(outputPath string, pages []pageResult) error {
	payload := map[string]interface{}{
		"pages": func() []map[string]interface{} {
			list := make([]map[string]interface{}, len(pages))
			for i, page := range pages {
				list[i] = map[string]interface{}{
					"index":        page.Index,
					"page_number":  page.Index + 1,
					"raw_text":     page.RawText,
					"markdown":     page.Markdown,
					"boxes":        page.Boxes,
					"image_assets": page.ImageAssets,
				}
			}
			return list
		}(),
	}
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()
	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(payload)
}

func shouldStore(name string) bool {
	lower := strings.ToLower(name)
	switch {
	case strings.HasSuffix(lower, ".jpg"),
		strings.HasSuffix(lower, ".jpeg"),
		strings.HasSuffix(lower, ".png"),
		strings.HasSuffix(lower, ".webp"),
		strings.HasSuffix(lower, ".gif"):
		return true
	default:
		return false
	}
}

func prepareArchiveHeader(entry archiveEntry) (*zip.FileHeader, error) {
	info, err := os.Stat(entry.Path)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("%s is a directory", entry.Path)
	}
	header := &zip.FileHeader{
		Name:   entry.Name,
		Method: entry.Method,
	}
	header.SetModTime(info.ModTime())
	header.UncompressedSize64 = uint64(info.Size())
	header.ExternalAttrs = (uint32(info.Mode().Perm()) & 0o777) << 16
	if entry.Method == zip.Store {
		f, err := os.Open(entry.Path)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		hash := crc32.NewIEEE()
		if _, err := io.Copy(hash, f); err != nil {
			return nil, err
		}
		header.CRC32 = hash.Sum32()
	}
	return header, nil
}

func writeArchive(archivePath string, entries []archiveEntry, progress func(done int, total int, name string)) error {
	if len(entries) == 0 {
		return nil
	}
	file, err := os.Create(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	zipWriter := zip.NewWriter(file)

	type job struct {
		index int
		entry archiveEntry
	}
	type result struct {
		index  int
		header *zip.FileHeader
		entry  archiveEntry
		err    error
	}

	tasks := make(chan job)
	results := make(chan result, len(entries))
	workerCount := minInt(maxInt(2, runtime.NumCPU()/2), len(entries))
	if workerCount <= 0 {
		workerCount = 1
	}
	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := range tasks {
				header, err := prepareArchiveHeader(task.entry)
				results <- result{
					index:  task.index,
					header: header,
					entry:  task.entry,
					err:    err,
				}
			}
		}()
	}
	go func() {
		wg.Wait()
		close(results)
	}()
	for idx, entry := range entries {
		tasks <- job{index: idx, entry: entry}
	}
	close(tasks)

	prepared := make([]result, len(entries))
	for res := range results {
		if res.err != nil {
			return res.err
		}
		prepared[res.index] = res
	}

	for i, item := range prepared {
		header := item.header
		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}
		source, err := os.Open(item.entry.Path)
		if err != nil {
			return err
		}
		if _, err := io.Copy(writer, source); err != nil {
			source.Close()
			return err
		}
		source.Close()
		if progress != nil {
			progress(i+1, len(entries), header.Name)
		}
	}

	if err := zipWriter.Close(); err != nil {
		return err
	}
	return nil
}

func ensureDefaultConfig(cfg *Config) {
	if cfg.MaxConcurrency <= 0 {
		cfg.MaxConcurrency = 2
	}
	if cfg.RenderWorkers < 0 {
		cfg.RenderWorkers = 0
	}
	if cfg.RequestTimeout <= 0 {
		cfg.RequestTimeout = 300
	}
}

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
	reportProgress := func(current int, message string) {
		total := int(atomic.LoadInt64(&progressTotal))
		if total < 1 {
			total = 1
		}
		writer.Progress(current, total, message)
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
		reportProgress(int(atomic.LoadInt64(&completed)), fmt.Sprintf("第 %d/%d 页已排队", job.index+1, totalPages))
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
			reportProgress(done, fmt.Sprintf("第 %d/%d 页识别完成", j.index+1, totalPages))
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

	archivePath := filepath.Join(cfg.OutputDir, "result.zip")
	baseProgress := pagesDone
	if err := writeArchive(archivePath, entries, func(done int, total int, name string) {
		reportProgress(baseProgress+done, fmt.Sprintf("正在压缩结果资源 (%d/%d)", done, total))
	}); err != nil {
		writer.Error(err)
		os.Exit(1)
	}

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
		},
	}
	payload["total_pages"] = totalPages
	writer.Result(payload)
}

func processPage(ctx context.Context, cfg Config, index int, imagePath string, imagesDir string) (pageResult, error) {
	pageImg, err := loadImage(imagePath)
	if err != nil {
		return pageResult{}, err
	}
	imageB64, err := encodeImageToBase64(imagePath)
	if err != nil {
		return pageResult{}, err
	}
	rawText, err := runInference(ctx, cfg, imageB64)
	if err != nil {
		return pageResult{}, err
	}
	markdown, assets, err := replaceDetectionBlocks(rawText, pageImg, index, imagesDir)
	if err != nil {
		return pageResult{}, err
	}
	boxes := parseDetections(rawText, pageImg.Bounds().Dx(), pageImg.Bounds().Dy())
	return pageResult{
		Index:       index,
		Markdown:    markdown,
		RawText:     rawText,
		ImageAssets: assets,
		Boxes:       boxes,
	}, nil
}
