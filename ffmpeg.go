package main

import (
	"fmt"
	"ipfs-livestream/cliexec"
	"path"
	"time"
)

type FFMpegController struct {
	ffmpeg string
	videoDevice string
	audioDevice string
	cliexec.Controller
}

func NewFFMpegController(ffmpegPath string) *FFMpegController {
	return &FFMpegController {
		ffmpeg: ffmpegPath,
		videoDevice: "1",
		audioDevice: "0",
		Controller: cliexec.Controller{},
	}
}

func (c *FFMpegController) RecordScreen(filename string, length time.Duration) error {
	data, err := c.ExecutePathWithDuration(c.ffmpeg, []string{"-f", "avfoundation", "-i",
	c.videoDevice + ":" + c.audioDevice, "-pix_fmt", "yuv420p", "-y", "-r", "10", filename},
	length)
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func (c *FFMpegController) ConvertVideo(filename, newExtension string) (string, error) {
	newFilename := path.Dir(filename) + "/" + path.Base(filename) + "." + newExtension
	data, err := c.ExecutePath(c.ffmpeg, []string{"-i", filename, newFilename})
	if err != nil {
		return newFilename, err
	}
	fmt.Println(string(data))
	return newFilename, nil
}