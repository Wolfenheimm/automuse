package main

import (
	"io"
	"log"

	"github.com/jonas747/dca"
)

// Encodes the video for audio playback
func (v *VoiceInstance) DCA(url string) {
	opts := dca.StdEncodeOptions
	opts.RawOutput = true
	opts.Bitrate = 94
	opts.Application = "lowdelay"

	encodeSession, err := dca.EncodeFile(url, opts)
	if err != nil {
		log.Println("FATA: Failed creating an encoding session: ", err)
	}

	v.encoder = encodeSession
	done := make(chan error)
	stream := dca.NewStream(encodeSession, v.voice, done)
	v.stream = stream
	dcaErr := <-done
	if dcaErr != nil && dcaErr != io.EOF {
		log.Println("FATA: An error occured", err)
	}
}
