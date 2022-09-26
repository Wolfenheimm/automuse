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
		searchList = searchList + " " + strconv.Itoa(index) + ". " + name + "\n"
		index = index + 1
		searchQueue = append(searchQueue, SongSearch{id, name})
	}

	searchRequested = true
	s.ChannelMessageSend(m.ChannelID, searchList)
	log.Println(searchList)
}

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
