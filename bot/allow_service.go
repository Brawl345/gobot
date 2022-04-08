package bot

import (
	"errors"

	"github.com/Brawl345/gobot/models"
	"github.com/Brawl345/gobot/utils"
	"golang.org/x/exp/slices"
	"gopkg.in/telebot.v3"
)

type allowService struct {
	allowedChats []int64
	chatService  models.ChatService
	userService  models.UserService
}

func NewAllowService(chatService models.ChatService, userService models.UserService) (*allowService, error) {
	allowedUsers, err := userService.GetAllAllowed()
	if err != nil {
		return nil, err
	}

	allowedChats, err := userService.GetAllAllowed()
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

func (service *allowService) IsUserAllowed(user *telebot.User) bool {
	if utils.IsAdmin(user) {
		return true
	}

	return slices.Contains(service.allowedChats, user.ID)
}

func (service *allowService) IsChatAllowed(chat *telebot.Chat) bool {
	return slices.Contains(service.allowedChats, chat.ID)
}

func (service *allowService) AllowUser(user *telebot.User) error {
	err := service.userService.Allow(user)
	if err != nil {
		return err
	}

	service.allowedChats = append(service.allowedChats, user.ID)
	return nil
}

func (service *allowService) DenyUser(user *telebot.User) error {
	if utils.IsAdmin(user) {
		return errors.New("cannot deny admin")
	}
	err := service.userService.Deny(user)
	if err != nil {
		return err
	}

	index := slices.Index(service.allowedChats, user.ID)
	service.allowedChats = slices.Delete(service.allowedChats, index, index+1)
	return nil
}

func (service *allowService) AllowChat(chat *telebot.Chat) error {
	err := service.chatService.Allow(chat)
	if err != nil {
		return err
	}

	service.allowedChats = append(service.allowedChats, chat.ID)
	return nil
}

func (service *allowService) DenyChat(chat *telebot.Chat) error {
	err := service.chatService.Deny(chat)
	if err != nil {
		return err
	}

	index := slices.Index(service.allowedChats, chat.ID)
	service.allowedChats = slices.Delete(service.allowedChats, index, index+1)
	return nil
}
