package main

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
