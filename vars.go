package main

import (
	"github.com/bwmarrin/discordgo"
	yt "github.com/kkdai/youtube/v2"
)

// Bot Parameters
var (
	botToken     string
	youtubeToken string
	s            *discordgo.Session
	v            = new(VoiceInstance)
	client       = yt.Client{Debug: true}
	song         = Song{}
	queue        = []Song{}
)
