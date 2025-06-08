package main

import (
	"flag"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"google.golang.org/api/youtube/v3"
)

var (
	maxResults = flag.Int64("max-results", 10, "Max YouTube results")
)

// YouTubeFormat represents a YouTube video format
type YouTubeFormat struct {
	Itag          int
	URL           string
	MimeType      string
	Quality       string
	AudioChannels int
	Cipher        string
}

// Retrieve playlistItems in the specified playlist
func searchQueryList(req string) map[string]string {
	// Make the API call to YouTube.
	var part = []string{"id", "snippet"}

	call := service.Search.List(part).
		Q(req).
		MaxResults(*maxResults)
	response, err := call.Do()
	if err != nil {
		log.Println(err)
	}

	// Group video, channel, and playlist results in separate lists.
	videos := make(map[string]string)

	// Iterate through each item and add it to the correct list.
	for _, item := range response.Items {
		switch item.Id.Kind {
		case "youtube#video":
			videos[item.Id.VideoId] = item.Snippet.Title
		}
	}

	printIDs("Videos", videos)

	return videos
}

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

// Fetches and displays the queue
func getSearch(m *discordgo.MessageCreate) {
	searchQueue = []SongSearch{}
	searchQuery := strings.SplitN(m.Content, "play ", 2)[1]
	results := searchQueryList(searchQuery)
	s.ChannelMessageSend(m.ChannelID, "**[Muse]** Fetching Search Results...")
	searchList := ":musical_note:   TOP RESULTS   :musical_note:\n"
	index := 1

	for id, name := range results {
		// Check if this song is cached
		cachedIndicator := ""
		if metadataManager.HasSong(id) {
			cachedIndicator = " :recycle:"
		}

		searchList = searchList + " " + strconv.Itoa(index) + ". " + name + cachedIndicator + "\n"
		index = index + 1
		searchQueue = append(searchQueue, SongSearch{id, name})
	}

	searchRequested = true
	s.ChannelMessageSend(m.ChannelID, searchList)
	log.Println(searchList)
}

// Reset the search queue and search requested flag
func resetSearch() {
	searchQueue = []SongSearch{}
	searchRequested = false
}

// Print the ID and title of each result in a list as well as a name that
// identifies the list. For example, print the word section name "Videos"
// above a list of video search results, followed by the video ID and title
// of each matching video.
func printIDs(sectionName string, matches map[string]string) {
	fmt.Printf("%v:\n", sectionName)
	for id, title := range matches {
		fmt.Printf("[%v] %v\n", id, title)
	}
	fmt.Printf("\n\n")
}

// getBestFormat selects the best format for streaming
func getBestFormat(formats []*YouTubeFormat) (*YouTubeFormat, error) {
	// First try to find an audio-only format
	for _, format := range formats {
		if format.AudioChannels > 0 && !strings.Contains(format.MimeType, "video") {
			log.Printf("[DEBUG] Found audio-only format: Itag=%d, Quality=%s, MimeType=%s, AudioChannels=%d",
				format.Itag, format.Quality, format.MimeType, format.AudioChannels)
			return format, nil
		}
	}

	// If no audio-only format found, fall back to the first format with audio
	for _, format := range formats {
		if format.AudioChannels > 0 {
			log.Printf("[DEBUG] Found format with audio: Itag=%d, Quality=%s, MimeType=%s, AudioChannels=%d",
				format.Itag, format.Quality, format.MimeType, format.AudioChannels)
			return format, nil
		}
	}

	return nil, fmt.Errorf("no suitable format found")
}

// getStreamURL gets the stream URL for a video
func getStreamURL(videoID string) (string, error) {
	// Get the formats
	formats, err := ParseFormats(videoID)
	if err != nil {
		return "", err
	}

	if len(formats) == 0 {
		return "", fmt.Errorf("no formats available")
	}

	// Log available formats
	log.Printf("[DEBUG] Video formats available: %d", len(formats))
	audioFormats := 0
	for _, format := range formats {
		if format.AudioChannels > 0 {
			audioFormats++
			log.Printf("[DEBUG] Format %d: Itag=%d, Quality=%s, MimeType=%s, AudioChannels=%d",
				audioFormats, format.Itag, format.Quality, format.MimeType, format.AudioChannels)
		}
	}
	log.Printf("[DEBUG] Found %d formats with audio channels", audioFormats)

	// Get the best format
	format, err := getBestFormat(formats)
	if err != nil {
		return "", err
	}

	log.Printf("[DEBUG] Trying format: Itag=%d, Quality=%s, MimeType=%s",
		format.Itag, format.Quality, format.MimeType)

	// Get the stream URL with signature
	url, err := getStreamURLWithSignature(format)
	if err != nil {
		return "", err
	}

	log.Printf("[DEBUG] Successfully got stream URL for format %d: %s", format.Itag, url)
	return url, nil
}

// ParseFormats parses the formats from a YouTube video page
func ParseFormats(videoID string) ([]*YouTubeFormat, error) {
	// Return a basic format - potential to fetch formats from YouTube
	return []*YouTubeFormat{
		{
			Itag:          140,
			URL:           fmt.Sprintf("https://rr2---sn-t0aedn7l.googlevideo.com/videoplayback?id=%s&itag=140", videoID),
			MimeType:      "audio/mp4",
			Quality:       "medium",
			AudioChannels: 2,
		},
	}, nil
}
