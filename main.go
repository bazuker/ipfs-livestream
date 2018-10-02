package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"time"
)

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

func main() {
	configPtr := flag.String("config", "config.json", "a path to configuration")
	samplesPtr := flag.Int("samples", -1, "max amount of stream samples (stream length)")
	watchPtr := flag.String("watch", "", "stream address to watch")

	flag.Parse()

	if configPtr == nil || !fileExists(*configPtr) {
		log.Println("invalid configuration file or path", *configPtr)
		return
	}

	hostAddrLen := 0

	if watchPtr != nil {
		hostAddrLen = len(*watchPtr)
		if hostAddrLen > 1 && hostAddrLen < 10 {
			log.Println("invalid stream address")
			return
		}
	}

	data, err := ioutil.ReadFile(*configPtr)
	if err != nil {
		log.Println("failed to read the file", *configPtr)
		return
	}

	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		log.Println("failed to unmarshal json of the file", *configPtr)
		return
	}

	log.Println("loaded configuration", config)
	if *samplesPtr == -1 {
		log.Println("stream length is unlimited")
	} else {
		log.Println("stream length is ", time.Duration(int(config.SampleDuration)*(*samplesPtr)))
	}

	stream := NewLivestream(config.FFmpeg, config.IPFS, config.SamplesPath, config.SampleDuration)
	err = stream.UseDefaultDevices() // use first devices from the list
	if err != nil {
		panic(err)
	}
	if watchPtr != nil && hostAddrLen > 1 {
		err = stream.Watch(*watchPtr)
	} else {
		err = stream.Broadcast(*samplesPtr)
	}
	if err != nil {
		panic(err)
	}
}
