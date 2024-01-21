package sql

import (
	"errors"
	"github.com/PaulSonOfLars/gotgbot/v2"

	"github.com/Brawl345/gobot/model"
	"github.com/Brawl345/gobot/utils"
	"golang.org/x/exp/slices"
)

type allowService struct {
	allowedChats []int64
	chatService  model.ChatService
	userService  model.UserService
}

func NewAllowService(chatService model.ChatService, userService model.UserService) (*allowService, error) {
	allowedUsers, err := userService.GetAllAllowed()
	if err != nil {
		return nil, err
	}

	allowedChats, err := chatService.GetAllAllowed()
	if err != nil {
		return nil, err
	}

	allowedChats = append(allowedChats, allowedUsers...)

	return &allowService{
		chatService:  chatService,
		userService:  userService,
		allowedChats: allowedChats,
	}, nil
}

func (service *allowService) IsUserAllowed(user *gotgbot.User) bool {
	if utils.IsAdmin(user) {
		return true
	}

	return slices.Contains(service.allowedChats, user.Id)
}

func (service *allowService) IsChatAllowed(chat *gotgbot.Chat) bool {
	return slices.Contains(service.allowedChats, chat.Id)
}

func (service *allowService) AllowUser(user *gotgbot.User) error {
	err := service.userService.Allow(user)
	if err != nil {
		return err
	}

	service.allowedChats = append(service.allowedChats, user.Id)
	return nil
}

func (service *allowService) DenyUser(user *gotgbot.User) error {
	if utils.IsAdmin(user) {
		return errors.New("cannot deny admin")
	}
	err := service.userService.Deny(user)
	if err != nil {
		return err
	}

	index := slices.Index(service.allowedChats, user.Id)
	service.allowedChats = slices.Delete(service.allowedChats, index, index+1)
	return nil
}

func (service *allowService) AllowChat(chat *gotgbot.Chat) error {
	err := service.chatService.Allow(chat)
	if err != nil {
		return err
	}

	service.allowedChats = append(service.allowedChats, chat.Id)
	return nil
}

func (service *allowService) DenyChat(chat *gotgbot.Chat) error {
	err := service.chatService.Deny(chat)
	if err != nil {
		return err
	}

	index := slices.Index(service.allowedChats, chat.Id)
	service.allowedChats = slices.Delete(service.allowedChats, index, index+1)
	return nil
}
