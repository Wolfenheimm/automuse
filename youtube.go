package main

import (
	"log"
	"net/http"

	"github.com/bwmarrin/discordgo"
	"google.golang.org/api/googleapi/transport"
	"google.golang.org/api/youtube/v3"
)

// Retrieve playlistItems in the specified playlist
func playlistItemsList(service *youtube.Service, part []string, playlistId string, pageToken string) *youtube.PlaylistItemListResponse {
	call := service.PlaylistItems.List(part)
	call = call.PlaylistId(playlistId)
	if pageToken != "" {
		call = call.PageToken(pageToken)
	}
	response, err := call.Do()
	log.Println(err)

	if err != nil {
		return &youtube.PlaylistItemListResponse{}
	} else {
		return response
	}
}

func queuePlaylist(playlistID string, m *discordgo.MessageCreate) {
	// This client is used to search playlists
	ytClient := &http.Client{
		Transport: &transport.APIKey{Key: youtubeToken},
	}

	service, err := youtube.New(ytClient)
	if err != nil {
		log.Fatalf("Error creating new YouTube client: %v", err)
	}

	nextPageToken := "" // Used to iterate through videos in a playlist

	for {
		// Retrieve next set of items in the playlist.
		var snippet = []string{"snippet"}
		playlistResponse := playlistItemsList(service, snippet, playlistID, nextPageToken)

		for _, playlistItem := range playlistResponse.Items {
			videoId := playlistItem.Snippet.ResourceId.VideoId
			content := "https://www.youtube.com/watch?v=" + videoId

			// Get Video Data
			video, err := client.GetVideo(content)
			if err != nil {
				log.Println(err)
			} else {
				format := video.Formats.WithAudioChannels() // Get matches with audio channels only
				song = fillSongInfo(m.ChannelID, m.Author.ID, m.ID, video.ID, video.Title, video.Duration.String())
				url, err := client.GetStreamURL(video, &format[0])

				if err != nil {
					log.Println(err)
				} else {
					song.VideoURL = url
					queue = append(queue, song)
				}
			}
		}

		// Set the token to retrieve the next page of results
		nextPageToken = playlistResponse.NextPageToken

		// Nothing left, break out
		if nextPageToken == "" {
			break
		}
	}
}
