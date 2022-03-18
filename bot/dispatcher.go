package bot

import (
	"gopkg.in/telebot.v3"
	"log"
)

func (bot *Nextbot) OnText(c telebot.Context) error {
	log.Printf("%s: %s", c.Chat().FirstName, c.Message().Text)

	isAllowed := bot.IsUserAllowed(c.Sender())
	if c.Message().FromGroup() && !isAllowed {
		isAllowed = bot.IsChatAllowed(c.Chat())
	}

	if !isAllowed {
		return nil
	}

	var err error

	if c.Message().Private() {
		err = bot.DB.Users.Create(c.Sender())
	} else {
		err = bot.DB.ChatsUsers.Create(c.Chat(), c.Sender())
	}
	if err != nil {
		return err
	}

	text := c.Message().Caption
	if text == "" {
		text = c.Message().Text
	}

	for _, plugin := range bot.plugins {
		for _, handler := range plugin.GetHandlers() {
			if !c.Message().FromGroup() && handler.GroupOnly {
				continue
			}

			matches := handler.Command.FindStringSubmatch(text)
			if len(matches) > 0 {
				log.Printf("Matched plugin %s: %s", plugin.GetName(), handler.Command)
				if bot.isPluginEnabled(plugin.GetName()) {
					if c.Message().FromGroup() && bot.isPluginDisabledForChat(c.Chat(), plugin.GetName()) {
						log.Printf("Plugin %s is disabled for this chat", plugin.GetName())
						continue
					}

					if handler.AdminOnly && !isAdmin(c.Sender()) {
						log.Println("User is not an admin.")
						continue
					}

					go handler.Handler(NextbotContext{
						Context: c,
						Matches: matches,
					})
				} else {
					log.Printf("Plugin %s is disabled globally", plugin.GetName())
				}
			}
		}
	}

	return nil
}

func (bot *Nextbot) OnUserJoined(c telebot.Context) error {
	return bot.DB.ChatsUsers.CreateBatch(c.Chat(), &c.Message().UsersJoined)
}

func (bot *Nextbot) OnUserLeft(c telebot.Context) error {
	if c.Message().UserLeft.IsBot {
		return nil
	}
	return bot.DB.ChatsUsers.Leave(c.Chat(), c.Message().UserLeft)
}
