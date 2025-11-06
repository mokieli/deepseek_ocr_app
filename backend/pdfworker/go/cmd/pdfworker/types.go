package main

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

