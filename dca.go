package main

import (
	"fmt"
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
	for err := range done {

		// Something horrible happened...
		if err != nil && err != io.EOF {
			log.Println("FATA: An error occured", err)
		}

		// Hit EOF, cleanup & stop
		if err == io.EOF {
			// Clean up incase something happened and ffmpeg is still running
			fmt.Printf("Should stop the song...")
			encodeSession.Cleanup()
			break
		}
	}
}
