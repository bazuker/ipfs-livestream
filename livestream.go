package main

import (
	"encoding/json"
	"github.com/go-playground/lars"
	"github.com/pkg/errors"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync/atomic"
	"time"
)

const syncFile = "sync.json"

type Livestream struct {
	Parts          []string      `json:"parts"`
	TempSample     string        `json:"-"`
	SampleCursor   int32         `json:"cursor"`
	SampleDuration time.Duration `json:"sample"`
	Ended          bool          `json:"ended"`
	Started        string        `json:"started"`
	Updated        string        `json:"updated"`

	dataFolder string `json:"-"`

	ipfsController   *IPFSController   `json:"-"`
	ffmpegController *FFMpegController `json:"-"`

	_sync          int32  `json:"-"`
	_lastSync      int32  `json:"-"`
	_syncfileCache []byte `json:"-"`
}

func NewLivestream(ffmpeg, ipfs, ipget, dataFolder string, sampleDuration time.Duration) *Livestream {
	return &Livestream{
		dataFolder:       dataFolder,
		Parts:            make([]string, 0),
		TempSample:       "",
		SampleCursor:     0,
		SampleDuration:   sampleDuration,
		Ended:            false,
		ipfsController:   NewIPFSController(ipfs, ipget),
		ffmpegController: NewFFMpegController(ffmpeg),
	}
}

func enableCors(c lars.Context) {
	c.Response().Header().Set("Access-Control-Allow-Origin", "*")
	c.Next()
}

func (ls *Livestream) watchSync(c lars.Context) {
	c.Response().Header().Set("Content-Type", "application/json")
	c.Response().Write(ls._syncfileCache)
	c.Response().WriteHeader(http.StatusOK)
}

func (ls *Livestream) UseDefaultDevices() error {
	devices, err := ls.ffmpegController.GetAvailableDevices()
	if err != nil {
		return err
	}
	if len(devices.Video) < 1 || len(devices.Audio) < 1 {
		return errors.New("video or audio device is unavailable")
	}
	ls.ffmpegController.videoDevice = devices.Video[0]
	ls.ffmpegController.audioDevice = devices.Audio[0]
	return nil
}

func (ls *Livestream) SetDevices(videoDevice, audioDevice string) {
	ls.ffmpegController.videoDevice = videoDevice
	ls.ffmpegController.audioDevice = audioDevice
}

func (ls *Livestream) Watch(address string) error {
	var err error
	log.Println("reading the stream", address)

	router := lars.New()
	router.Use(enableCors)
	router.Get("/sync", ls.watchSync)
	server := &http.Server{Addr: ":8888", Handler: router.Serve()}

	go server.ListenAndServe()
	defer server.Close()

	lastHash := ""
	for !ls.Ended {
		log.Println("checking for updates...")
		syncPath := ls.dataFolder + "/" + syncFile
		err = ls.ipfsController.GetResource(syncPath, address, true)
		if err != nil {
			return err
		}
		hash, err := hashMD5(syncPath)
		if err != nil {
			return err
		}
		if hash == lastHash {
			log.Println("no updates from the streamer")
			time.Sleep(ls.SampleDuration)
			continue
		}
		lastHash = hash
		data, err := ioutil.ReadFile(syncPath)
		if err != nil {
			return err
		}
		ls._syncfileCache = data
		err = json.Unmarshal(data, ls)
		if err != nil {
			return err
		}
		log.Println("stream updated. Now contains", len(ls.Parts), "parts")
	}
	log.Println("stream ended")
	return nil
}

func (ls *Livestream) Broadcast(samples int) error {
	var err error
	createDir(ls.dataFolder)
	id, err := ls.ipfsController.GetId()
	if err != nil {
		return err
	}
	log.Println("Broadcasting with ID", id.ID)
	ls.Started = time.Now().String()
	i := 0
	for {
		if samples > 0 {
			i++
			if i > samples {
				syncCursor := atomic.LoadInt32(&ls._sync)
				if syncCursor > 0 {
					ls._lastSync = syncCursor
					log.Println("waiting for the synchronization to finish...")
					time.Sleep(time.Second * 5)
					continue
				} else if ls._lastSync != ls.SampleCursor {
					log.Println("running the final synchronization...")
					ls.Ended = true
					ls.safeSync()
				}
				return nil
			}
		}
		if ls.SampleCursor > 0 {
			if !fileExists(ls.TempSample) {
				return errors.New("sample does not exist or was not recorded")
			}
			go ls.pushSample(ls.TempSample)
		}
		// record the screen
		ls.TempSample = ls.dataFolder + "/sample_" + strconv.Itoa(int(ls.SampleCursor)) + ".mp4"
		log.Println("recording...", ls.TempSample)
		err = ls.recordSample()
		if err != nil {
			return err
		}
		ls.SampleCursor++
	}
}

func (ls *Livestream) recordSample() error {
	return ls.ffmpegController.RecordScreen(ls.TempSample, ls.SampleDuration)
}

func (ls *Livestream) pushSample(tempSample string) {
	const tenSecond = time.Second * 10
	t := ls.SampleDuration / 2
	if t > tenSecond {
		log.Println("preparing in", tenSecond)
		time.Sleep(tenSecond)
	} else {
		log.Println("preparing in", t)
		time.Sleep(t)
	}
	// adding to IPFS
	log.Println("uploading...")
	fn, err := ls.ipfsController.PushFile(tempSample)
	if err != nil {
		panic(err)
	}
	log.Println("added", fn)
	ls.Parts = append(ls.Parts, fn)
	// update the stream
	ls.safeSync()
}

func (ls *Livestream) safeSync() {
	log.Println("synchronizing...")
	data, err := json.Marshal(ls)
	if err != nil {
		log.Println("failed to encode the sync.json due", err.Error())
		return
	}
	err = ioutil.WriteFile(ls.dataFolder+"/"+syncFile, data, os.ModePerm)
	if err != nil {
		log.Println("failed to write to sync.json due", err.Error())
		return
	}
	err = ls.sync()
	if err != nil {
		log.Println("ERROR:", err.Error())
	}
}

func (ls *Livestream) sync() error {
	if atomic.LoadInt32(&ls._sync) > 0 {
		log.Println("aborted. Awaiting for the previous synchronization to finish")
		return nil
	}
	ls.Updated = time.Now().String()
	atomic.StoreInt32(&ls._sync, ls.SampleCursor)
	defer atomic.StoreInt32(&ls._sync, 0)
	hash, err := ls.ipfsController.PushFile(ls.dataFolder + "/" + syncFile)
	if err != nil {
		return err
	}
	err = ls.ipfsController.PublishName(hash)
	log.Println("synchronization is over for", hash)
	return err
}
