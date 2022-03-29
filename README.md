# AutoMuse
Automuse is a discord bot that plays music in a discord voice channel via commands. At the moment, only youtube links can be played.

:point_right: You can add this bot to your server [here](https://discord.com/api/oauth2/authorize?client_id=955836104559460362&permissions=534723950656&scope=bot%20applications.commands)

# Requirements
- GoLang 1.18
- Your very own bot token placed in an environment variable (See Link Above)
     - Env var: BOT_TOKEN

# How to use
- Typing the play command in any text channel will trigger the bot to join your voice channel, you must be in a voice channel for this to work.
- Playing additional links will place the songs in a queue. 
- The queue will auto-play until done.
- You can skip songs in the queue.
- At 0 songs left in the queue the bot will leave the channel, play a new song to bring it back in.

## Syntax
###### Base Commands to Use the Bot
````
play https://www.youtube.com/watch?v=<VIDEO-ID>
skip
stop
````