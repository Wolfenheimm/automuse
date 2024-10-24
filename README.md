# AutoMuse
Automuse is a discord bot that plays YouTube music in a discord voice channel via commands.

The bot is still a WIP and has some kinks from time to time.

:point_right: Set up your bot on the discord developer portal [here](https://discord.com/developers/applications), and see the permissions section below for more information on how to configure it. You will also need to Get/set your bot token from here as well.

:point_right: Follow the official YouTube documentation to get/set your YouTube API key [here](https://developers.google.com/youtube/v3/docs)

## Permissions

Automuse requires the `bot` **scope** and several permissions on a server to work properly. Therefore, ensure to set these in the developer portal during the creation of the invite link:
- `Send Messages`
- `Connect`
- `Speak`

# Requirements
- GoLang 1.23
- A Linux environment:
    - A [Discord Bot Token](https://discord.com/developers/applications) placed in an environment variable
        - Env var: BOT_TOKEN
    - A [YouTube API Key]((https://developers.google.com/youtube/v3/docs)) placed in an environment variable
        - Env var: YT_TOKEN
    - Your Discord Guild ID and Channel ID placed in environment variables
        - Env var: GUILD_ID
        - Env var: GENERAL_CHAT_ID <-- You may choose any voice channel in your server
        - Tip: Enable Developer Mode in Discord to get this information
    - The Discord Bot will require the following OAuth2 permissions:
        - Scope:
            - Bot
        - Permissions:
            - Connect
            - Speak
            - Send Messages
    

# How to use
- Run the project from within its directory - `go run .`
- You may only use YouTube links & it doesn't necessarily have to be a song
- You must be in a voice channel in order to play content from YT
- Adding the -pl argument will play a YT playlist in its entirety, provided the URL is a public playlist
- Playing additional content while one is playing will place the content in a queue

## Syntax
###### Base Commands to Use the Bot
````
play https://www.youtube.com/watch?v=<VIDEO-ID>                         -> Plays/Queues a video(audio)
play https://www.youtube.com/playlist?list=<PLAYLIST-ID>                -> Plays/Queues a playlist
play -pl https://www.youtube.com/watch?v=<VIDEO-ID>&list=<PLAYLIST-ID>  -> Plays/Queues a playlist
play #                                                                  -> Plays a video(audio) from the queue & skips song playing
play your search query string                                           -> Shows a list of videos, prompt: play # after to queue
skip                                                                    -> Skips the current Song
skip to #                                                               -> Skips to a specific song in the playlist
stop                                                                    -> Stops the current song and clears the queue
queue                                                                   -> Shows the current queue in chat
remove #                                                                -> Remove a song from queue at number #
````