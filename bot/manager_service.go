package bot

import (
	"errors"

	"github.com/Brawl345/gobot/models"
	"github.com/Brawl345/gobot/plugin"
	"golang.org/x/exp/slices"
	"gopkg.in/telebot.v3"
)

type managerService struct {
	chatsPluginsService    models.ChatsPluginsService
	pluginService          models.PluginService
	plugins                []plugin.Plugin
	enabledPlugins         []string
	disabledPluginsForChat map[int64][]string
}

func NewManagerService(
	chatsPluginsService models.ChatsPluginsService,
	pluginService models.PluginService,
) (*managerService, error) {

	enabledPlugins, err := pluginService.GetAllEnabled()
	if err != nil {
		return nil, err
	}

	disabledPluginsForChat, err := chatsPluginsService.GetAllDisabled()
	if err != nil {
		return nil, err
	}

	return &managerService{
		chatsPluginsService:    chatsPluginsService,
		pluginService:          pluginService,
		enabledPlugins:         enabledPlugins,
		disabledPluginsForChat: disabledPluginsForChat,
	}, nil
}

func (service *managerService) SetPlugins(plugins []plugin.Plugin) {
	service.plugins = plugins
}

func (service *managerService) EnablePlugin(name string) error {
	if slices.Contains(service.enabledPlugins, name) {
		return errors.New("✅ Das Plugin ist bereits aktiv")
	}

	for _, plg := range service.plugins {
		if plg.Name() == name {
			err := service.pluginService.Enable(name)
			if err != nil {
				return err
			}
			service.enabledPlugins = append(service.enabledPlugins, name)
			return nil
		}
	}
	return errors.New("❌ Plugin existiert nicht")
}

func (service *managerService) isPluginDisabledForChat(chat *telebot.Chat, name string) bool {
	disabledPlugins, exists := service.disabledPluginsForChat[chat.ID]
	if !exists {
		return false
	}
	return slices.Contains(disabledPlugins, name)
}

func (service *managerService) EnablePluginForChat(chat *telebot.Chat, name string) error {
	if !service.isPluginDisabledForChat(chat, name) {
		return errors.New("✅ Das Plugin ist für diesen Chat schon aktiv")
	}

	for _, plg := range service.plugins {
		if plg.Name() == name {
			err := service.chatsPluginsService.Enable(chat, name)
			if err != nil {
				return err
			}

			index := slices.Index(service.disabledPluginsForChat[chat.ID], name)
			service.disabledPluginsForChat[chat.ID] = slices.Delete(service.disabledPluginsForChat[chat.ID],
				index, index+1)

			return nil
		}
	}
	return errors.New("❌ Plugin existiert nicht")
}

func (service *managerService) DisablePlugin(name string) error {
	if !slices.Contains(service.enabledPlugins, name) {
		return errors.New("✅ Das Plugin ist nicht aktiv")
	}

	err := service.pluginService.Disable(name)
	if err != nil {
		return err
	}
	index := slices.Index(service.enabledPlugins, name)
	service.enabledPlugins = slices.Delete(service.enabledPlugins, index, index+1)
	return nil
}

func (service *managerService) DisablePluginForChat(chat *telebot.Chat, name string) error {
	if service.isPluginDisabledForChat(chat, name) {
		return errors.New("✅ Das Plugin ist für diesen Chat schon deaktiviert")
	}

	for _, plg := range service.plugins {
		if plg.Name() == name {
			err := service.chatsPluginsService.Disable(chat, name)
			if err != nil {
				return err
			}

			service.disabledPluginsForChat[chat.ID] = append(service.disabledPluginsForChat[chat.ID], name)

			return nil
		}
	}
	return errors.New("❌ Plugin existiert nicht")
}

func (service *managerService) isPluginEnabled(name string) bool {
	return slices.Contains(service.enabledPlugins, name)
}
