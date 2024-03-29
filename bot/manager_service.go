package bot

import (
	"github.com/Brawl345/gobot/model"
	"github.com/Brawl345/gobot/plugin"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"golang.org/x/exp/slices"
)

type managerService struct {
	chatsPluginsService    model.ChatsPluginsService
	pluginService          model.PluginService
	plugins                []plugin.Plugin
	enabledPlugins         []string
	disabledPluginsForChat map[int64][]string
}

func NewManagerService(
	chatsPluginsService model.ChatsPluginsService,
	pluginService model.PluginService,
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

func (service *managerService) Plugins() []plugin.Plugin {
	return service.plugins
}

func (service *managerService) SetPlugins(plugins []plugin.Plugin) {
	service.plugins = plugins
}

func (service *managerService) EnablePlugin(name string) error {
	if slices.Contains(service.enabledPlugins, name) {
		return model.ErrAlreadyExists
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
	return model.ErrNotFound
}

func (service *managerService) IsPluginDisabledForChat(chat *gotgbot.Chat, name string) bool {
	disabledPlugins, exists := service.disabledPluginsForChat[chat.Id]
	if !exists {
		return false
	}
	return slices.Contains(disabledPlugins, name)
}

func (service *managerService) EnablePluginForChat(chat *gotgbot.Chat, name string) error {
	if !service.IsPluginDisabledForChat(chat, name) {
		return model.ErrAlreadyExists
	}

	for _, plg := range service.plugins {
		if plg.Name() == name {
			err := service.chatsPluginsService.Enable(chat, name)
			if err != nil {
				return err
			}

			index := slices.Index(service.disabledPluginsForChat[chat.Id], name)
			service.disabledPluginsForChat[chat.Id] = slices.Delete(service.disabledPluginsForChat[chat.Id],
				index, index+1)

			return nil
		}
	}
	return model.ErrNotFound
}

func (service *managerService) DisablePlugin(name string) error {
	if !slices.Contains(service.enabledPlugins, name) {
		return model.ErrNotFound
	}

	err := service.pluginService.Disable(name)
	if err != nil {
		return err
	}
	index := slices.Index(service.enabledPlugins, name)
	service.enabledPlugins = slices.Delete(service.enabledPlugins, index, index+1)
	return nil
}

func (service *managerService) DisablePluginForChat(chat *gotgbot.Chat, name string) error {
	if service.IsPluginDisabledForChat(chat, name) {
		return model.ErrAlreadyExists
	}

	for _, plg := range service.plugins {
		if plg.Name() == name {
			err := service.chatsPluginsService.Disable(chat, name)
			if err != nil {
				return err
			}

			service.disabledPluginsForChat[chat.Id] = append(service.disabledPluginsForChat[chat.Id], name)

			return nil
		}
	}
	return model.ErrNotFound
}

func (service *managerService) IsPluginEnabled(name string) bool {
	return slices.Contains(service.enabledPlugins, name)
}
