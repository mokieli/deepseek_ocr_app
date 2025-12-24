package main

type Config struct {
	TaskID           string `json:"task_id"`
	PDFPath          string `json:"pdf_path"`
	OutputDir        string `json:"output_dir"`
	DPI              int    `json:"dpi"`
	Prompt           string `json:"prompt"`
	InferURL         string `json:"infer_url"`
	AuthToken        string `json:"auth_token"`
	BaseSize         int    `json:"base_size"`
	ImageSize        int    `json:"image_size"`
	CropMode         bool   `json:"crop_mode"`
	MaxConcurrency   int    `json:"max_concurrency"`
	RenderWorkers    int    `json:"render_workers"`
	RequestTimeout   int    `json:"request_timeout_seconds"`
	OriginalFilename string `json:"original_filename"`
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

