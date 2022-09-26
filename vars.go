package main

import (
	"context"

	"github.com/bwmarrin/discordgo"
	"github.com/jonas747/dca"
	yt "github.com/kkdai/youtube/v2"
	"google.golang.org/api/youtube/v3"
)

// Bot Parameters
var (
	botToken        string
	youtubeToken    string
	searchRequested bool
	service         *youtube.Service
	s               *discordgo.Session
	v               = new(VoiceInstance)
	opts            = dca.StdEncodeOptions
	client          = yt.Client{Debug: true}
	ctx             = context.Background()
	song            = Song{}
	searchQueue     = []SongSearch{}
	queue           = []Song{}
	badQualitySongs = BadQualitySongNodes{}
)
