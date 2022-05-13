package bot

import (
	"os"
	"sort"
	"strings"
	"time"

	"github.com/Brawl345/gobot/logger"
	"github.com/Brawl345/gobot/models/sql"
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
	"github.com/Brawl345/gobot/plugin/getfile"
	"github.com/Brawl345/gobot/plugin/google_images"
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
	"github.com/Brawl345/gobot/plugin/stats"
	"github.com/Brawl345/gobot/plugin/twitter"
	"github.com/Brawl345/gobot/plugin/upload_by_url"
	"github.com/Brawl345/gobot/plugin/urbandictionary"
	"github.com/Brawl345/gobot/plugin/weather"
	"github.com/Brawl345/gobot/plugin/wikipedia"
	"github.com/Brawl345/gobot/plugin/worldclock"
	"github.com/Brawl345/gobot/plugin/youtube"
	"gopkg.in/telebot.v3"
)

var log = logger.New("bot")

type (
	Gobot struct {
		Telebot *telebot.Bot
	}
)

func New() (*Gobot, error) {
	db, err := sql.New()
	if err != nil {
		return nil, err
	}

	bot, err := telebot.NewBot(telebot.Settings{
		Token:  strings.TrimSpace(os.Getenv("BOT_TOKEN")),
		Poller: GetPoller(),
	})
	if err != nil {
		return nil, err
	}

	// Calling "remove webook" even if no webhook is set so pending updates can be dropped
	err = bot.RemoveWebhook(true)
	if err != nil {
		return nil, err
	}

	// General services
	chatService := sql.NewChatService(db)
	credentialService := sql.NewCredentialService(db)
	geocodingService := sql.NewGeocodingService()
	pluginService := sql.NewPluginService(db)
	userService := sql.NewUserService(db)
	chatsPluginsService := sql.NewChatsPluginsService(db, chatService, pluginService)
	chatsUsersService := sql.NewChatsUsersService(db, chatService, userService)

	// Plugin-specific services
	afkService := sql.NewAfkService(db)
	birthdayService := sql.NewBirthdayService(db)
	cleverbotService := sql.NewCleverbotService(db)
	fileService := sql.NewFileService(db)
	googleImagesService := sql.NewGoogleImagesService(db)
	googleImagesCleanupService := sql.NewGoogleImagesCleanupService(db)
	homeService := sql.NewHomeService(db)
	notifyService := sql.NewNotifyService(db)
	quoteService := sql.NewQuoteService(db)
	randomService := sql.NewRandomService(db)
	reminderService := sql.NewReminderService(db)
	rkiService := sql.NewRKIService(db)

	allowService, err := sql.NewAllowService(chatService, userService)
	if err != nil {
		return nil, err
	}

	managerService, err := NewManagerService(chatsPluginsService, pluginService)
	if err != nil {
		return nil, err
	}

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
		getfile.New(credentialService, fileService),
		google_images.New(credentialService, googleImagesService, googleImagesCleanupService),
		gps.New(geocodingService),
		home.New(geocodingService, homeService),
		id.New(),
		ids.New(chatsUsersService),
		kaomoji.New(),
		manager.New(managerService),
		myanimelist.New(credentialService),
		notify.New(notifyService),
		quotes.New(quoteService),
		randoms.New(randomService),
		reminders.New(bot, reminderService),
		replace.New(),
		rki.New(rkiService),
		stats.New(chatsUsersService),
		twitter.New(credentialService),
		upload_by_url.New(),
		urbandictionary.New(),
		weather.New(geocodingService, homeService),
		wikipedia.New(),
		worldclock.New(credentialService),
		youtube.New(credentialService),
	}
	managerService.SetPlugins(plugins)

	log.Info().Msgf("Loaded %d plugins", len(plugins))

	var commands []telebot.Command
	for _, plg := range plugins {
		commands = append(commands, plg.Commands()...)
	}
	sort.Slice(commands, func(i, j int) bool {
		return commands[i].Text < commands[j].Text
	})
	if commands != nil {
		if len(commands) > 100 {
			log.Warn().Msg("Too many commands, some will be ignored")
			commands = commands[:100]
		}
		err = bot.SetCommands(commands)
		if err != nil {
			log.Err(err).Msg("Failed to set commands")
		}
	}

	b := &Gobot{
		Telebot: bot,
	}

	d := &Dispatcher{
		allowService:      allowService,
		chatsUsersService: chatsUsersService,
		managerService:    managerService,
		userService:       userService,
	}

	_, shouldPrintMsgs := os.LookupEnv("PRINT_MSGS")
	if shouldPrintMsgs {
		b.Telebot.Use(PrintMessage)
	}

	b.Telebot.Handle(telebot.OnText, d.OnText)
	b.Telebot.Handle(telebot.OnEdited, d.OnText)
	b.Telebot.Handle(telebot.OnMedia, d.OnText)
	b.Telebot.Handle(telebot.OnContact, d.OnText)
	b.Telebot.Handle(telebot.OnLocation, d.OnText)
	b.Telebot.Handle(telebot.OnVenue, d.OnText)
	b.Telebot.Handle(telebot.OnGame, d.OnText)
	b.Telebot.Handle(telebot.OnDice, d.OnText)
	b.Telebot.Handle(telebot.OnUserJoined, d.OnUserJoined)
	b.Telebot.Handle(telebot.OnUserLeft, d.OnUserLeft)
	b.Telebot.Handle(telebot.OnCallback, d.OnCallback)
	b.Telebot.Handle(telebot.OnQuery, d.OnInlineQuery)

	b.Telebot.Handle(telebot.OnPinned, d.NullRoute)
	b.Telebot.Handle(telebot.OnNewGroupTitle, d.NullRoute)
	b.Telebot.Handle(telebot.OnNewGroupPhoto, d.NullRoute)
	b.Telebot.Handle(telebot.OnGroupPhotoDeleted, d.NullRoute)
	b.Telebot.Handle(telebot.OnGroupCreated, d.NullRoute)

	b.Telebot.OnError = OnError

	return b, nil
}

func GetPoller() telebot.Poller {
	allowedUpdates := []string{"message", "edited_message", "callback_query", "inline_query"}

	webhookPort := strings.TrimSpace(os.Getenv("PORT"))
	webhookURL := strings.TrimSpace(os.Getenv("WEBHOOK_PUBLIC_URL"))

	if webhookPort == "" || webhookURL == "" {
		log.Debug().Msg("Using long polling")
		return &telebot.LongPoller{
			AllowedUpdates: allowedUpdates,
			Timeout:        10 * time.Second,
		}
	}

	log.Debug().
		Str("port", webhookPort).
		Str("webhook_public_url", webhookURL).
		Msg("Using webhook")

	return &telebot.Webhook{
		Listen:         ":" + webhookPort,
		AllowedUpdates: allowedUpdates,
		MaxConnections: 50,
		DropUpdates:    true,
		Endpoint: &telebot.WebhookEndpoint{
			PublicURL: webhookURL,
		},
	}
}
