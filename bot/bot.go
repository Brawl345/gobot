package bot

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/model/sql"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/plugin/about"
	"github.com/Brawl345/gobot/plugin/afk"
	"github.com/Brawl345/gobot/plugin/alive"
	"github.com/Brawl345/gobot/plugin/allow"
	"github.com/Brawl345/gobot/plugin/amazon_ref_cleaner"
	"github.com/Brawl345/gobot/plugin/birthdays"
	"github.com/Brawl345/gobot/plugin/calc"
	"github.com/Brawl345/gobot/plugin/cleverbot"
	"github.com/Brawl345/gobot/plugin/covid"
	"github.com/Brawl345/gobot/plugin/creds"
	"github.com/Brawl345/gobot/plugin/currency"
	"github.com/Brawl345/gobot/plugin/dcrypt"
	"github.com/Brawl345/gobot/plugin/delmsg"
	"github.com/Brawl345/gobot/plugin/echo"
	"github.com/Brawl345/gobot/plugin/expand"
	"github.com/Brawl345/gobot/plugin/gemini"
	"github.com/Brawl345/gobot/plugin/getfile"
	"github.com/Brawl345/gobot/plugin/google_images"
	"github.com/Brawl345/gobot/plugin/google_search"
	"github.com/Brawl345/gobot/plugin/gps"
	"github.com/Brawl345/gobot/plugin/home"
	"github.com/Brawl345/gobot/plugin/id"
	"github.com/Brawl345/gobot/plugin/ids"
	"github.com/Brawl345/gobot/plugin/kaomoji"
	"github.com/Brawl345/gobot/plugin/manager"
	"github.com/Brawl345/gobot/plugin/myanimelist"
	"github.com/Brawl345/gobot/plugin/notify"
	"github.com/Brawl345/gobot/plugin/quotes"
	"github.com/Brawl345/gobot/plugin/randoms"
	"github.com/Brawl345/gobot/plugin/reminders"
	"github.com/Brawl345/gobot/plugin/replace"
	"github.com/Brawl345/gobot/plugin/rki"
	"github.com/Brawl345/gobot/plugin/speech_to_text"
	"github.com/Brawl345/gobot/plugin/stats"
	"github.com/Brawl345/gobot/plugin/summarize"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/jmoiron/sqlx"
)

var log = logger.New("bot")

type (
	Gobot struct {
		GoTgBot *gotgbot.Bot
		updater *ext.Updater
	}
)

func New(db *sqlx.DB) (*Gobot, error) {
	// General services
	chatService := sql.NewChatService(db)
	credentialService := sql.NewCredentialService(db)
	geocodingService := sql.NewGeocodingService()
	pluginService := sql.NewPluginService(db)
	userService := sql.NewUserService(db)
	chatsPluginsService := sql.NewChatsPluginsService(db, chatService, pluginService)
	chatsUsersService := sql.NewChatsUsersService(db, chatService, userService)
	allowService, err := sql.NewAllowService(chatService, userService)
	if err != nil {
		return nil, err
	}
	managerSrvce, err := NewManagerService(chatsPluginsService, pluginService)
	if err != nil {
		return nil, err
	}

	// Bot itself
	bot, err := gotgbot.NewBot(strings.TrimSpace(os.Getenv("BOT_TOKEN")), nil)
	if err != nil {
		return nil, err
	}

	dispatcher := ext.NewDispatcher(&ext.DispatcherOpts{
		Processor: NewProcessor(allowService, chatsUsersService, managerSrvce, userService),
	})
	updater := ext.NewUpdater(dispatcher, &ext.UpdaterOpts{
		UnhandledErrFunc: OnError,
	})

	// Plugin-specific services
	afkService := sql.NewAfkService(db)
	birthdayService := sql.NewBirthdayService(db)
	cleverbotService := sql.NewCleverbotService(db)
	fileService := sql.NewFileService(db)
	geminiService := sql.NewGeminiService(db)
	googleImagesService := sql.NewGoogleImagesService(db)
	googleImagesCleanupService := sql.NewGoogleImagesCleanupService(db)
	homeService := sql.NewHomeService(db)
	notifyService := sql.NewNotifyService(db)
	quoteService := sql.NewQuoteService(db)
	randomService := sql.NewRandomService(db)
	reminderService := sql.NewReminderService(db)
	rkiService := sql.NewRKIService(db)

	plugins := []plugin.Plugin{
		about.New(),
		afk.New(afkService),
		alive.New(),
		allow.New(allowService),
		amazon_ref_cleaner.New(),
		birthdays.New(bot, birthdayService),
		calc.New(),
		cleverbot.New(credentialService, cleverbotService),
		covid.New(),
		creds.New(credentialService),
		currency.New(),
		dcrypt.New(),
		delmsg.New(),
		echo.New(),
		expand.New(),
		gemini.New(credentialService, geminiService),
		getfile.New(credentialService, fileService),
		google_images.New(credentialService, googleImagesService, googleImagesCleanupService),
		google_search.New(credentialService),
		gps.New(geocodingService),
		home.New(geocodingService, homeService),
		id.New(),
		ids.New(chatsUsersService),
		kaomoji.New(),
		manager.New(managerSrvce),
		myanimelist.New(credentialService),
		notify.New(notifyService),
		quotes.New(quoteService),
		randoms.New(randomService),
		reminders.New(bot, reminderService),
		replace.New(),
		rki.New(rkiService),
		speech_to_text.New(credentialService),
		stats.New(chatsUsersService),
		summarize.New(credentialService),
		//twitter.New(),
		//upload_by_url.New(),
		//urbandictionary.New(),
		//weather.New(geocodingService, homeService),
		//wikipedia.New(),
		//worldclock.New(credentialService, geocodingService),
		//youtube.New(credentialService),
	}
	managerSrvce.SetPlugins(plugins)

	log.Info().Msgf("Loaded %d plugins", len(plugins))

	var commands []gotgbot.BotCommand
	for _, plg := range plugins {
		commands = append(commands, plg.Commands()...)
	}
	sort.Slice(commands, func(i, j int) bool {
		return commands[i].Command < commands[j].Command
	})
	if commands != nil {
		if len(commands) > 100 {
			log.Warn().Msg("Too many commands, some will be ignored")
			commands = commands[:100]
		}
		_, err = bot.SetMyCommands(commands, nil)
		if err != nil {
			log.Err(err).Msg("Failed to set commands")
		}
	}

	webhookPort := strings.TrimSpace(os.Getenv("PORT"))
	webhookURL := strings.TrimSpace(os.Getenv("WEBHOOK_PUBLIC_URL"))
	webhookUrlPath := os.Getenv("WEBHOOK_URL_PATH")

	allowedUpdates := []string{"message", "edited_message", "callback_query", "inline_query"}

	if webhookPort == "" || webhookURL == "" || webhookUrlPath == "" {
		log.Debug().Msg("Using long polling")
		err = updater.StartPolling(bot, &ext.PollingOpts{
			DropPendingUpdates: true,
			GetUpdatesOpts: &gotgbot.GetUpdatesOpts{
				AllowedUpdates: allowedUpdates,
				Timeout:        10,
				RequestOpts: &gotgbot.RequestOpts{
					Timeout: time.Second * 15,
				},
			},
		})
		if err != nil {
			return nil, err
		}
	} else {
		log.Debug().
			Str("port", webhookPort).
			Str("webhook_public_url", webhookURL).
			Str("webhook_url_path", webhookUrlPath).
			Msg("Using webhook")

		webhookSecret := strings.TrimSpace(os.Getenv("WEBHOOK_SECRET"))
		if webhookSecret == "" {
			log.Warn().Msg("WEBHOOK_SECRET not set, it's STRONGLY RECOMMENDED to set one!")
		}

		webhookOpts := ext.WebhookOpts{
			ListenAddr:  fmt.Sprintf(":%s", webhookPort),
			SecretToken: webhookSecret,
		}
		err = updater.StartWebhook(bot, webhookUrlPath, webhookOpts)
		if err != nil {
			return nil, err
		}

		ok, err := bot.SetWebhook(webhookURL, &gotgbot.SetWebhookOpts{
			AllowedUpdates:     allowedUpdates,
			MaxConnections:     50,
			DropPendingUpdates: true,
			SecretToken:        webhookSecret,
		})
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, fmt.Errorf("failed to set webhook")
		}
	}

	b := &Gobot{
		GoTgBot: bot,
		updater: updater,
	}

	return b, nil
}

func (b *Gobot) Start() {
	b.updater.Idle()
}
