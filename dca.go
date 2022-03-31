package main

import (
	"io"
	"log"

	"github.com/jonas747/dca"
)

func (v *VoiceInstance) DCA(url string) {
	opts := dca.StdEncodeOptions
	opts.RawOutput = true
	opts.Bitrate = 128
	opts.Application = "lowdelay"

	encodeSession, err := dca.EncodeFile(url, opts)
	if err != nil {
		log.Println("FATA: Failed creating an encoding session: ", err)
	}

	v.encoder = encodeSession
	done := make(chan error)
	stream := dca.NewStream(encodeSession, v.voice, done)
	v.stream = stream
	for {
		select {
		case err := <-done:
			if err != nil && err != io.EOF {
				log.Println("FATA: An error occured", err)
			}
			// Clean up incase something happened and ffmpeg is still running
			encodeSession.Cleanup()
			return
		}
	}
}
