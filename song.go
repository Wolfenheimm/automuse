package main

// Fill a song struct - Used for the song queue
func fillSongInfo(channelID string, authorID string, Id string, videoID string, title string, duration string) (songData Song) {
	// Fill Song Info
	song = Song{
		ChannelID: channelID,
		User:      authorID,
		ID:        Id,
		VidID:     videoID,
		Title:     title,
		Duration:  duration,
		VideoURL:  "",
	}

	return song
}
