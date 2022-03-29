package main

func SearchGuild(textChannelID string) (guildID string) {
	channel, _ := s.Channel(textChannelID)
	guildID = channel.GuildID
	return
}

func SearchVoiceChannel(user string) (voiceChannelID string) {
	for _, g := range s.State.Guilds {
		for _, v := range g.VoiceStates {
			if v.UserID == user {
				return v.ChannelID
			}
		}
	}
	return ""
}
