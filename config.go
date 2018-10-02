package main

import "time"

type Config struct {
	FFmpeg         string        `json:"ffmpeg"`
	IPFS           string        `json:"ipfs"`
	SamplesPath    string        `json:"samples_path"`
	SampleDuration time.Duration `json:"sample_duration"`
}
