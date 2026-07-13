package sql

import (
	"errors"
	"sync"

	"github.com/Brawl345/gobot/utils/tgUtils"
	"github.com/PaulSonOfLars/gotgbot/v2"

	"github.com/Brawl345/gobot/model"
	"slices"
)

type allowService struct {
	mu           sync.RWMutex
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
	if tgUtils.IsAdmin(user) {
		return true
	}

	service.mu.RLock()
	defer service.mu.RUnlock()
	return slices.Contains(service.allowedChats, user.Id)
}

func (service *allowService) IsChatAllowed(chat *gotgbot.Chat) bool {
	service.mu.RLock()
	defer service.mu.RUnlock()
	return slices.Contains(service.allowedChats, chat.Id)
}

func (service *allowService) AllowUser(user *gotgbot.User) error {
	service.mu.Lock()
	defer service.mu.Unlock()

	err := service.userService.Allow(user)
	if err != nil {
		return err
	}

	service.allowedChats = append(service.allowedChats, user.Id)
	return nil
}

func (service *allowService) DenyUser(user *gotgbot.User) error {
	if tgUtils.IsAdmin(user) {
		return errors.New("cannot deny admin")
	}

	service.mu.Lock()
	defer service.mu.Unlock()

	err := service.userService.Deny(user)
	if err != nil {
		return err
	}

	index := slices.Index(service.allowedChats, user.Id)
	if index >= 0 {
		service.allowedChats = slices.Delete(service.allowedChats, index, index+1)
	}
	return nil
}

func (service *allowService) AllowChat(chat *gotgbot.Chat) error {
	service.mu.Lock()
	defer service.mu.Unlock()

	err := service.chatService.Allow(chat)
	if err != nil {
		return err
	}

	service.allowedChats = append(service.allowedChats, chat.Id)
	return nil
}

func (service *allowService) DenyChat(chat *gotgbot.Chat) error {
	service.mu.Lock()
	defer service.mu.Unlock()

	err := service.chatService.Deny(chat)
	if err != nil {
		return err
	}

	index := slices.Index(service.allowedChats, chat.Id)
	if index >= 0 {
		service.allowedChats = slices.Delete(service.allowedChats, index, index+1)
	}
	return nil
}
