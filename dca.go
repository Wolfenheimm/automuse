package main

import (
	"io"
	"log"

	"github.com/jonas747/dca"
)

// Encodes the video for audio playback
func (v *VoiceInstance) DCA(url string) {
	encodeSession, err := dca.EncodeFile(url, opts)
	if err != nil {
		log.Println("FATA: Failed creating an encoding session: ", err)
		return
	}

	v.encoder = encodeSession
	done := make(chan error)
	stream := dca.NewStream(encodeSession, v.voice, done)
	v.stream = stream
	dcaErr := <-done
	if dcaErr != nil && dcaErr != io.EOF {
		log.Println("DCA stopped suddenly: ", dcaErr)
	}
}

func setUpDcaOptions() {
	opts.RawOutput = true
	opts.Bitrate = 96
	opts.BufferedFrames = 50
	opts.Application = "lowdelay"
}
