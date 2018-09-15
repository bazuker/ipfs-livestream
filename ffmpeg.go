package main

import (
	"errors"
	"fmt"
	"ipfs-livestream/cliexec"
	"path"
	"runtime"
	"strings"
	"time"
)

type FFMpegController struct {
	ffmpeg      string
	videoDevice string
	audioDevice string
	cliexec.Controller
}

type Devices struct {
	Video []string
	Audio []string
}

func NewFFMpegController(ffmpegPath string) *FFMpegController {
	return &FFMpegController{
		ffmpeg:      ffmpegPath,
		videoDevice: "1",
		audioDevice: "0",
		Controller:  cliexec.Controller{},
	}
}

func (c *FFMpegController) RecordScreen(filename string, length time.Duration) error {
	var params []string
	if runtime.GOOS == "windows" {
		params = []string{"-hide_banner", "-y", "-rtbufsize", "200M", "-f", "gdigrab", "-thread_queue_size", "1024", "-probesize", "10M", "-r", "30", "-draw_mouse",
			"1", "-i", "desktop", "-f", "dshow", "-channel_layout", "stereo", "-thread_queue_size", "1024", "-i", "audio=" + c.audioDevice, "-c:v",
			"libx264", "-r", "30", "-preset", "ultrafast", "-tune", "zerolatency", "-crf", "25", "-pix_fmt", "yuv420p", "-c:a", "aac", "-strict", "-2", "-ac", "2", "-b:a", "128k", filename}
	} else {
		params = []string{"-f", "avfoundation", "-i", c.videoDevice + ":" + c.audioDevice, "-pix_fmt", "yuv420p", "-y", "-r", "10", filename}
	}
	_, err := c.ExecutePathWithDuration(c.ffmpeg, params, length)
	return err
}

func (c *FFMpegController) GetAvailableDevices() (*Devices, error) {
	if runtime.GOOS == "windows" {
		const immediateExit = "Immediate exit requested"
		const immediateExitLen = len(immediateExit)
		data, err := c.ExecutePath(c.ffmpeg, []string{"-hide_banner", "-list_devices", "true", "-f", "dshow", "-i", "dummy"})
		output := strings.TrimSpace(string(data))
		if err != nil {
			// for some reason "immediate exit requested" is interpreted as error on windows
			// so we have to work around that
			size := len(output)
			if size < immediateExitLen || output[size-immediateExitLen:] != immediateExit {
				return nil, err
			}
		}
		devices := &Devices{make([]string, 0), make([]string, 0)}
		// now a little bit of black magic to parse the output of ffmpeg
		lines := strings.Split(output, "[")
		videoList := false
		for _, line := range lines {
			if len(line) > 2 {
				clean := strings.TrimSpace(line[strings.IndexRune(line, ']')+1:])
				if strings.HasPrefix(clean, "DirectShow video devices") {
					videoList = true
				} else if strings.HasSuffix(clean, "DirectShow audio devices") {
					videoList = false
				} else if !strings.HasPrefix(clean, "Alternative name") {
					if videoList {
						devices.Video = append(devices.Video, clean)
						continue
					}
					devices.Audio = append(devices.Audio, clean)
				}
			}
		}
		return devices, nil
	}

	if runtime.GOOS == "darwin" {
		data, _ := c.ExecutePath(c.ffmpeg, []string{"-hide_banner", "-f", "avfoundation", "-list_devices", "true", "-i", ""})
		output := strings.TrimSpace(string(data))

		devices := &Devices{make([]string, 0), make([]string, 0)}
		// now a little bit of black magic to parse the output of ffmpeg
		videoList := false
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			clean := strings.TrimSpace(line[strings.IndexRune(line, ']')+1:])
			if clean == "AVFoundation video devices:" {
				videoList = true
			} else if clean == "AVFoundation audio devices:" {
				videoList = false
			} else {
				if videoList {
					devices.Video = append(devices.Video, clean)
					continue
				}
				devices.Audio = append(devices.Audio, clean)
			}
		}
		return devices, nil
	}

	return nil, errors.New("unsupported os")
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
