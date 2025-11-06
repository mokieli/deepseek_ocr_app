package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

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
			continue
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
