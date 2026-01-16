package pkg

func (c *Command) SaySlur() {
	//if util.messageInChannel(c.Session, c.MessageEvent, "general") {
	//	util.SendMessageWithError(c.Session, c.MessageEvent, "I can't say slurs in general anymore :x:", "refusing to say a slur")
	//	c.Session.ChannelMessageSendEmbed(c.MessageEvent.ChannelID, &discordgo.MessageEmbed{
	//		Title: util.GetRandomEmbedTitle(),
	//		Image: &discordgo.MessageEmbedImage{
	//			URL:    "https://static.wikia.nocookie.net/nicos-nextbots-fanmade/images/f/f6/1984.png/revision/latest?cb=20240210060355",
	//			Height: 1920,
	//			Width:  1080,
	//		},
	//	})
	//	return
	//
	//}
	//
	//resp, err := http.Get("https://gist.githubusercontent.com/Vizdun/0e9d76834d609dde09842be9bab53db7/raw/71116ec3446288aea56bd52a228f54881568844e/rsdb.json")
	//if err != nil {
	//	_, err = c.Session.ChannelMessageSend(c.MessageEvent.ChannelID, fmt.Sprintf("failed to download slur database: %v", err))
	//	return
	//}
	//
	//var slurs []Slur
	//
	//b, err := io.ReadAll(resp.Body)
	//resp.Body.Close()
	//
	//err = json.Unmarshal(b, &slurs)
	//if err != nil {
	//	_, err = c.Session.ChannelMessageSend(c.MessageEvent.ChannelID, fmt.Sprintf("failed to unmarshal slur database: %v", err))
	//	return
	//}
	//
	//slur := slurs[rand.Intn(len(slurs))]
	//content := fmt.Sprintf("Heres a slur: %s. group: %s. description: %s", slur.Slur, slur.Group, slur.Desc)
	//c.Logger.Info(content)
	//
	//go c.TTS.GenerateAndPlay(c.Session, c.MessageEvent, content, c.TTSOpts.ChannelName)
	//
	//msg := fmt.Sprintf("||```\n%s\n```||", content)
	//util.SendMessageWithError(c.Session, c.MessageEvent, msg, "failed to send message")
}

type Slur struct {
	Slur  string `json:"slur"`
	Group string `json:"group"`
	Desc  string `json:"desc"`
}
