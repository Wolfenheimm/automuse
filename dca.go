package main

import (
	"io"
	"log"
	"strings"

	"github.com/jonas747/dca"
)

// Encodes the video for audio playback
func (v *VoiceInstance) DCA(path string, isMpeg bool) {
	log.Println("INFO: Streaming from URL:", v.nowPlaying.VideoURL)
	log.Println("INFO: Starting DCA function")

	var encodeSession *dca.EncodeSession
	var err error

	if isMpeg {
		dirPath := "mpegs/" + path
		log.Println("INFO: Encoding MPEG file at path:", dirPath)
		encodeSession, err = dca.EncodeFile(dirPath, opts)
	} else {
		log.Println("INFO: Encoding file at path:", path)
		encodeSession, err = dca.EncodeFile(path, opts)
	}

	if err != nil {
		log.Println("FATA: Failed creating an encoding session: ", err)
		return
	}
	defer encodeSession.Cleanup()

	v.encoder = encodeSession
	done := make(chan error)
	stream := dca.NewStream(encodeSession, v.voice, done)
	v.stream = stream

	log.Println("INFO: DCA stream started")

	dcaErr := <-done
	log.Println("INFO: DCA stream finished")

	if dcaErr != nil {
		log.Println("ERROR: DCA stream error: ", dcaErr) // Log the error message

		if dcaErr == io.EOF {
			log.Println("INFO: DCA stream ended normally with EOF")
		} else if strings.Contains(dcaErr.Error(), `strconv.ParseFloat: parsing "N": invalid syntax`) {
			log.Println("WARN: Invalid duration detected, setting duration to 0 to prevent crash.")
			v.nowPlaying.Duration = "0"
		} else {
			log.Println("DCA stopped suddenly: ", dcaErr)
		}
		v.stream = nil
	} else {
		log.Println("INFO: DCA stream ended without error")
	}
}

func setUpDcaOptions() {
	opts.RawOutput = false
	opts.Bitrate = 128
	opts.Application = "lowdelay"
	opts.Volume = 100
	opts.CompressionLevel = 10
	opts.FrameDuration = 20
	opts.PacketLoss = 1
	opts.VBR = true
	opts.BufferedFrames = 10000
}
