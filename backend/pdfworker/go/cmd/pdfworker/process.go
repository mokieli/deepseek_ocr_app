package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

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
	if strings.TrimSpace(rawText) == "" {
		fmt.Fprintf(os.Stderr, "pdfworker notice: empty OCR text (task=%s page=%d image=%s)\n", cfg.TaskID, index, filepath.Base(imagePath))
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

