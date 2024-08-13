# AutoMuse
Automuse is a discord bot that plays music in a discord voice channel via commands. At the moment, only youtube video or playlist links can be played. It's also possible to query youtube instead of entering links and choose from a menu. Feel free to add songs to the songQualityIssues.json file, I take requests and add them in!

The bot is still a WIP and may not work as intended.

:point_right: Get/set your bot token [here](https://discord.com/developers/applications/)

:point_right: Follow the official YouTube documentation to get/set your YouTube API key [here](https://developers.google.com/youtube/v3/docs)

:point_right: Set up your bot on the discord developer portal [here](https://discord.com/developers/applications), and see the permissions section below for more information on how to configure it.

## Permissions

Automuse requires the `bot` **scope** and several permissions on a server to work properly. Therefore, ensure to set these in the developer portal during the creation of the invite link:
- `Send Messages`
- `Connect`
- `Speak`

# Requirements
- GoLang 1.19
- A Discord bot token placed in an environment variable (See Link Above)
     - Env var: BOT_TOKEN
- A YouTube API Key placed in an environment variable (See Link Above)
    - Env var: YT_TOKEN
- Your Discord Guild ID and Channel ID placed in environment variables
    - Env var: GUILD_ID
    - Env var: GENERAL_CHAT_ID <-- You may choose any voice channel in your server

# How to use
- You may only use YouTube links
- If you are experiencing sound quality issues, add them to the songQualityIssues.json file and choose a format
- Typing the play command in any text channel will trigger the bot to join your voice channel, you must be in a voice channel for this to work.
- Adding the -pl argument will play the playlist or the playlist the video is associated to
- Playing additional links will place the songs in a queue. 
- The queue will auto-play until done.
- You can skip songs in the queue.
- You can play a specific song from the queue.
- You can stop the current song and clear the queue.
- When no songs are left in the queue, the bot will leave the channel. Play a new song to bring it back in your voice channel.

## Syntax
###### Base Commands to Use the Bot
````
play https://www.youtube.com/watch?v=<VIDEO-ID>                         -> Plays/Queues a video(audio)
play https://www.youtube.com/playlist?list=<PLAYLIST-ID>                -> Plays/Queues a playlist
play -pl https://www.youtube.com/watch?v=<VIDEO-ID>&list=<PLAYLIST-ID>  -> Plays/Queues a playlist
play #                                                                  -> Plays a video(audio) from the queue & skips song playing
                                                                        -> Plays a video(audio) from the search & adds to queue
play your search query string                                           -> Shows a list of videos, play # after to queue
skip                                                                    -> Skips the current Song
skip to #                                                               -> Skips to a specific song in the playlist
stop                                                                    -> Stops the current song and clears the queue
queue                                                                   -> Shows the current queue in chat
remove #                                                                -> Remove a song from queue at number #
````