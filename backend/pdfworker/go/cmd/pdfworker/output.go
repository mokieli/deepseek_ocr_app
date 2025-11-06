package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
)

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

