package bot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/Brawl345/gobot/model"
	"github.com/Brawl345/gobot/plugin"
	"github.com/Brawl345/gobot/utils/tgUtils"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/jmoiron/sqlx"
)

const (
	testAdminID = int64(1234)
	testUserID  = int64(777)
)

func TestMain(m *testing.M) {
	_ = os.Setenv("ADMIN_ID", fmt.Sprintf("%d", testAdminID))
	_ = os.Unsetenv("PRINT_MSGS")
	os.Exit(m.Run())
}

type fakeAllowService struct {
	userAllowed bool
	chatAllowed bool
}

func (f *fakeAllowService) AllowChat(*gotgbot.Chat) error         { return nil }
func (f *fakeAllowService) AllowUser(*gotgbot.User) error         { return nil }
func (f *fakeAllowService) DenyChat(*gotgbot.Chat) error          { return nil }
func (f *fakeAllowService) DenyUser(*gotgbot.User) error          { return nil }
func (f *fakeAllowService) IsChatAllowed(*gotgbot.Chat) bool      { return f.chatAllowed }
func (f *fakeAllowService) IsUserAllowed(user *gotgbot.User) bool { return f.userAllowed }

type fakeChatsUsersService struct {
	created     []int64
	batches     [][]gotgbot.User
	left        []int64
	createError error
}

func (f *fakeChatsUsersService) Create(chat *gotgbot.Chat, _ *gotgbot.User) error {
	f.created = append(f.created, chat.Id)
	return f.createError
}

func (f *fakeChatsUsersService) CreateBatch(_ *gotgbot.Chat, users *[]gotgbot.User) error {
	f.batches = append(f.batches, *users)
	return nil
}

func (f *fakeChatsUsersService) GetAllUsersWithMsgCount(*gotgbot.Chat) ([]model.User, error) {
	return nil, nil
}

func (f *fakeChatsUsersService) IsAllowed(*gotgbot.Chat, *gotgbot.User) bool { return true }

func (f *fakeChatsUsersService) Leave(_ *gotgbot.Chat, user *gotgbot.User) error {
	f.left = append(f.left, user.Id)
	return nil
}

type fakeUserService struct {
	created []int64
}

func (f *fakeUserService) Allow(*gotgbot.User) error { return nil }

func (f *fakeUserService) Create(user *gotgbot.User) error {
	f.created = append(f.created, user.Id)
	return nil
}

func (f *fakeUserService) CreateTx(*sqlx.Tx, *gotgbot.User) error { return nil }
func (f *fakeUserService) Deny(*gotgbot.User) error               { return nil }
func (f *fakeUserService) GetAllAllowed() ([]int64, error)        { return nil, nil }

type fakeManagerService struct {
	plugins          []plugin.Plugin
	disabledGlobally map[string]bool
	disabledForChat  map[string]bool
}

func (f *fakeManagerService) Plugins() []plugin.Plugin                        { return f.plugins }
func (f *fakeManagerService) EnablePlugin(string) error                       { return nil }
func (f *fakeManagerService) EnablePluginForChat(*gotgbot.Chat, string) error { return nil }
func (f *fakeManagerService) DisablePlugin(string) error                      { return nil }
func (f *fakeManagerService) DisablePluginForChat(*gotgbot.Chat, string) error {
	return nil
}
func (f *fakeManagerService) IsPluginEnabled(name string) bool { return !f.disabledGlobally[name] }
func (f *fakeManagerService) IsPluginDisabledForChat(_ *gotgbot.Chat, name string) bool {
	return f.disabledForChat[name]
}

type fakePlugin struct {
	name     string
	handlers []plugin.Handler
}

func (p *fakePlugin) Name() string                            { return p.name }
func (p *fakePlugin) Commands() []gotgbot.BotCommand          { return nil }
func (p *fakePlugin) Handlers(*gotgbot.User) []plugin.Handler { return p.handlers }

type apiRequest struct {
	method string
	params map[string]any
}

type fakeBotClient struct {
	requests chan apiRequest
}

func newFakeBotClient() *fakeBotClient {
	return &fakeBotClient{requests: make(chan apiRequest, 16)}
}

func (f *fakeBotClient) RequestWithContext(_ context.Context, _ string, method string, params map[string]any, _ *gotgbot.RequestOpts) (json.RawMessage, error) {
	f.requests <- apiRequest{method: method, params: params}
	if method == "sendMessage" {
		return json.RawMessage(`{"message_id":1,"date":1,"chat":{"id":1,"type":"private"}}`), nil
	}
	return json.RawMessage(`true`), nil
}

func (f *fakeBotClient) GetAPIURL(*gotgbot.RequestOpts) string { return gotgbot.DefaultAPIURL }

func (f *fakeBotClient) FileURL(string, string, *gotgbot.RequestOpts) string { return "" }

type dispatchRecord struct {
	matches      []string
	namedMatches map[string]string
}

type testEnv struct {
	processor  *Processor
	bot        *gotgbot.Bot
	client     *fakeBotClient
	allow      *fakeAllowService
	manager    *fakeManagerService
	users      *fakeUserService
	chatsUsers *fakeChatsUsersService
}

func newTestEnv(plugins ...plugin.Plugin) *testEnv {
	allow := &fakeAllowService{userAllowed: true, chatAllowed: true}
	manager := &fakeManagerService{
		plugins:          plugins,
		disabledGlobally: map[string]bool{},
		disabledForChat:  map[string]bool{},
	}
	users := &fakeUserService{}
	chatsUsers := &fakeChatsUsersService{}
	client := newFakeBotClient()
	bot := &gotgbot.Bot{
		Token:     "test-token",
		User:      gotgbot.User{Id: 42, IsBot: true, FirstName: "Test", Username: "testbot"},
		BotClient: client,
	}
	return &testEnv{
		processor:  NewProcessor(allow, chatsUsers, manager, users),
		bot:        bot,
		client:     client,
		allow:      allow,
		manager:    manager,
		users:      users,
		chatsUsers: chatsUsers,
	}
}

func (e *testEnv) process(t *testing.T, update *gotgbot.Update) {
	t.Helper()
	ctx := ext.NewContext(e.bot, update, nil)
	if err := e.processor.ProcessUpdate(nil, e.bot, ctx); err != nil {
		t.Fatalf("ProcessUpdate returned error: %v", err)
	}
}

func commandHandler(trigger any, dispatched chan dispatchRecord) *plugin.CommandHandler {
	return &plugin.CommandHandler{
		Trigger: trigger,
		HandlerFunc: func(_ *gotgbot.Bot, c plugin.GobotContext) error {
			dispatched <- dispatchRecord{matches: c.Matches, namedMatches: c.NamedMatches}
			return nil
		},
	}
}

func expectDispatch(t *testing.T, ch chan dispatchRecord) dispatchRecord {
	t.Helper()
	select {
	case r := <-ch:
		return r
	case <-time.After(3 * time.Second):
		t.Fatal("handler was not dispatched")
		return dispatchRecord{}
	}
}

func expectNoDispatch(t *testing.T, ch chan dispatchRecord) {
	t.Helper()
	select {
	case <-ch:
		t.Fatal("handler was dispatched unexpectedly")
	case <-time.After(150 * time.Millisecond):
	}
}

func expectRequest(t *testing.T, client *fakeBotClient, method string) apiRequest {
	t.Helper()
	select {
	case r := <-client.requests:
		if r.method != method {
			t.Fatalf("expected API request %q, got %q", method, r.method)
		}
		return r
	case <-time.After(3 * time.Second):
		t.Fatalf("expected API request %q, got none", method)
		return apiRequest{}
	}
}

func privateChat() gotgbot.Chat {
	return gotgbot.Chat{Id: 100, Type: gotgbot.ChatTypePrivate}
}

func groupChat() gotgbot.Chat {
	return gotgbot.Chat{Id: -200, Type: gotgbot.ChatTypeSupergroup}
}

func textMessage(chat gotgbot.Chat, text string) *gotgbot.Message {
	return &gotgbot.Message{
		MessageId: 1,
		Date:      time.Now().Unix(),
		Text:      text,
		Chat:      chat,
		From:      &gotgbot.User{Id: testUserID, FirstName: "Tester"},
	}
}

func messageUpdate(msg *gotgbot.Message) *gotgbot.Update {
	return &gotgbot.Update{UpdateId: 1, Message: msg}
}

func callbackUpdate(data string, chat gotgbot.Chat, msgDate int64) *gotgbot.Update {
	return &gotgbot.Update{
		UpdateId: 2,
		CallbackQuery: &gotgbot.CallbackQuery{
			Id:   "cb1",
			From: gotgbot.User{Id: testUserID, FirstName: "Tester"},
			Data: data,
			Message: gotgbot.Message{
				MessageId: 5,
				Date:      msgDate,
				Chat:      chat,
			},
		},
	}
}

func inlineQueryUpdate(query string) *gotgbot.Update {
	return &gotgbot.Update{
		UpdateId: 3,
		InlineQuery: &gotgbot.InlineQuery{
			Id:    "iq1",
			From:  gotgbot.User{Id: testUserID, FirstName: "Tester"},
			Query: query,
		},
	}
}

func TestRegexpCommandDispatch(t *testing.T) {
	dispatched := make(chan dispatchRecord, 8)
	env := newTestEnv(&fakePlugin{
		name:     "start",
		handlers: []plugin.Handler{commandHandler(regexp.MustCompile(`^/start (?P<arg>\w+)$`), dispatched)},
	})

	env.process(t, messageUpdate(textMessage(privateChat(), "/start foo")))

	r := expectDispatch(t, dispatched)
	if len(r.matches) != 2 || r.matches[1] != "foo" {
		t.Errorf("unexpected matches: %v", r.matches)
	}
	if r.namedMatches["arg"] != "foo" {
		t.Errorf("unexpected named matches: %v", r.namedMatches)
	}
	if len(env.users.created) != 1 || env.users.created[0] != testUserID {
		t.Errorf("expected user %d to be tracked, got %v", testUserID, env.users.created)
	}
}

func TestRegexpCommandNoMatch(t *testing.T) {
	dispatched := make(chan dispatchRecord, 8)
	env := newTestEnv(&fakePlugin{
		name:     "start",
		handlers: []plugin.Handler{commandHandler(regexp.MustCompile(`^/start$`), dispatched)},
	})

	env.process(t, messageUpdate(textMessage(privateChat(), "hello world")))

	expectNoDispatch(t, dispatched)
}

func TestMultiplePluginsMatchSameMessage(t *testing.T) {
	dispatchedA := make(chan dispatchRecord, 8)
	dispatchedB := make(chan dispatchRecord, 8)
	env := newTestEnv(
		&fakePlugin{name: "a", handlers: []plugin.Handler{commandHandler(regexp.MustCompile(`multi`), dispatchedA)}},
		&fakePlugin{name: "b", handlers: []plugin.Handler{commandHandler(regexp.MustCompile(`multi`), dispatchedB)}},
	)

	env.process(t, messageUpdate(textMessage(privateChat(), "multi")))

	expectDispatch(t, dispatchedA)
	expectDispatch(t, dispatchedB)
}

func TestNotAllowed(t *testing.T) {
	dispatched := make(chan dispatchRecord, 8)
	env := newTestEnv(&fakePlugin{
		name:     "start",
		handlers: []plugin.Handler{commandHandler(regexp.MustCompile(`.*`), dispatched)},
	})
	env.allow.userAllowed = false
	env.allow.chatAllowed = false

	env.process(t, messageUpdate(textMessage(privateChat(), "/start")))

	expectNoDispatch(t, dispatched)
	if len(env.users.created) != 0 {
		t.Errorf("user should not be tracked, got %v", env.users.created)
	}
}

func TestGroupAllowedViaChat(t *testing.T) {
	dispatched := make(chan dispatchRecord, 8)
	env := newTestEnv(&fakePlugin{
		name:     "start",
		handlers: []plugin.Handler{commandHandler(regexp.MustCompile(`^/start$`), dispatched)},
	})
	env.allow.userAllowed = false
	env.allow.chatAllowed = true

	env.process(t, messageUpdate(textMessage(groupChat(), "/start")))

	expectDispatch(t, dispatched)
	if len(env.chatsUsers.created) != 1 || env.chatsUsers.created[0] != groupChat().Id {
		t.Errorf("expected chat-user tracking for chat %d, got %v", groupChat().Id, env.chatsUsers.created)
	}
}

func TestChatAllowanceIgnoredInPrivate(t *testing.T) {
	dispatched := make(chan dispatchRecord, 8)
	env := newTestEnv(&fakePlugin{
		name:     "start",
		handlers: []plugin.Handler{commandHandler(regexp.MustCompile(`^/start$`), dispatched)},
	})
	env.allow.userAllowed = false
	env.allow.chatAllowed = true

	env.process(t, messageUpdate(textMessage(privateChat(), "/start")))

	expectNoDispatch(t, dispatched)
}

func TestEditedMessage(t *testing.T) {
	withEdits := make(chan dispatchRecord, 8)
	withoutEdits := make(chan dispatchRecord, 8)

	editHandler := commandHandler(regexp.MustCompile(`^/edit$`), withEdits)
	editHandler.HandleEdits = true

	env := newTestEnv(
		&fakePlugin{name: "edits", handlers: []plugin.Handler{editHandler}},
		&fakePlugin{name: "noedits", handlers: []plugin.Handler{commandHandler(regexp.MustCompile(`^/edit$`), withoutEdits)}},
	)

	msg := textMessage(privateChat(), "/edit")
	msg.EditDate = time.Now().Unix()
	env.process(t, &gotgbot.Update{UpdateId: 1, EditedMessage: msg})

	expectDispatch(t, withEdits)
	expectNoDispatch(t, withoutEdits)
	if len(env.users.created) != 0 {
		t.Errorf("edited messages must not track users, got %v", env.users.created)
	}
}

func TestGroupOnly(t *testing.T) {
	dispatched := make(chan dispatchRecord, 8)
	handler := commandHandler(regexp.MustCompile(`^/group$`), dispatched)
	handler.GroupOnly = true
	env := newTestEnv(&fakePlugin{name: "group", handlers: []plugin.Handler{handler}})

	env.process(t, messageUpdate(textMessage(privateChat(), "/group")))
	expectNoDispatch(t, dispatched)

	env.process(t, messageUpdate(textMessage(groupChat(), "/group")))
	expectDispatch(t, dispatched)
}

func TestPluginDisabledGlobally(t *testing.T) {
	dispatched := make(chan dispatchRecord, 8)
	env := newTestEnv(&fakePlugin{
		name:     "start",
		handlers: []plugin.Handler{commandHandler(regexp.MustCompile(`^/start$`), dispatched)},
	})
	env.manager.disabledGlobally["start"] = true

	env.process(t, messageUpdate(textMessage(privateChat(), "/start")))

	expectNoDispatch(t, dispatched)
}

func TestPluginDisabledForChat(t *testing.T) {
	dispatched := make(chan dispatchRecord, 8)
	env := newTestEnv(&fakePlugin{
		name:     "start",
		handlers: []plugin.Handler{commandHandler(regexp.MustCompile(`^/start$`), dispatched)},
	})
	env.manager.disabledForChat["start"] = true

	env.process(t, messageUpdate(textMessage(groupChat(), "/start")))
	expectNoDispatch(t, dispatched)

	env.process(t, messageUpdate(textMessage(privateChat(), "/start")))
	expectDispatch(t, dispatched)
}

func TestPluginDisableToggleBetweenUpdates(t *testing.T) {
	dispatched := make(chan dispatchRecord, 8)
	env := newTestEnv(&fakePlugin{
		name:     "start",
		handlers: []plugin.Handler{commandHandler(regexp.MustCompile(`^/start$`), dispatched)},
	})

	env.process(t, messageUpdate(textMessage(privateChat(), "/start")))
	expectDispatch(t, dispatched)

	env.manager.disabledGlobally["start"] = true
	env.process(t, messageUpdate(textMessage(privateChat(), "/start")))
	expectNoDispatch(t, dispatched)

	env.manager.disabledGlobally["start"] = false
	env.process(t, messageUpdate(textMessage(privateChat(), "/start")))
	expectDispatch(t, dispatched)
}

func TestAdminOnlyCommand(t *testing.T) {
	dispatched := make(chan dispatchRecord, 8)
	handler := commandHandler(regexp.MustCompile(`^/admin$`), dispatched)
	handler.AdminOnly = true
	env := newTestEnv(&fakePlugin{name: "admin", handlers: []plugin.Handler{handler}})

	env.process(t, messageUpdate(textMessage(privateChat(), "/admin")))
	expectNoDispatch(t, dispatched)

	msg := textMessage(privateChat(), "/admin")
	msg.From = &gotgbot.User{Id: testAdminID, FirstName: "Admin"}
	env.process(t, messageUpdate(msg))
	expectDispatch(t, dispatched)
}

func TestMediaTriggers(t *testing.T) {
	photo := []gotgbot.PhotoSize{{FileId: "p1", Width: 1, Height: 1}}

	cases := []struct {
		name    string
		trigger tgUtils.MessageTrigger
		message func() *gotgbot.Message
		want    bool
	}{
		{"photo matches PhotoMsg", tgUtils.PhotoMsg, func() *gotgbot.Message {
			msg := textMessage(privateChat(), "")
			msg.Photo = photo
			return msg
		}, true},
		{"photo matches AnyMedia", tgUtils.AnyMedia, func() *gotgbot.Message {
			msg := textMessage(privateChat(), "")
			msg.Photo = photo
			return msg
		}, true},
		{"photo does not match DocumentMsg", tgUtils.DocumentMsg, func() *gotgbot.Message {
			msg := textMessage(privateChat(), "")
			msg.Photo = photo
			return msg
		}, false},
		{"document matches DocumentMsg", tgUtils.DocumentMsg, func() *gotgbot.Message {
			msg := textMessage(privateChat(), "")
			msg.Document = &gotgbot.Document{FileId: "d1"}
			return msg
		}, true},
		{"voice matches VoiceMsg", tgUtils.VoiceMsg, func() *gotgbot.Message {
			msg := textMessage(privateChat(), "")
			msg.Voice = &gotgbot.Voice{FileId: "v1"}
			return msg
		}, true},
		{"location matches LocationMsg", tgUtils.LocationMsg, func() *gotgbot.Message {
			msg := textMessage(privateChat(), "")
			msg.Location = &gotgbot.Location{Latitude: 1, Longitude: 2}
			return msg
		}, true},
		{"venue matches VenueMsg", tgUtils.VenueMsg, func() *gotgbot.Message {
			msg := textMessage(privateChat(), "")
			msg.Venue = &gotgbot.Venue{Title: "Somewhere"}
			return msg
		}, true},
		{"text matches AnyMsg", tgUtils.AnyMsg, func() *gotgbot.Message {
			return textMessage(privateChat(), "hello")
		}, true},
		{"photo matches AnyMsg", tgUtils.AnyMsg, func() *gotgbot.Message {
			msg := textMessage(privateChat(), "")
			msg.Photo = photo
			return msg
		}, true},
		{"text does not match AnyMedia", tgUtils.AnyMedia, func() *gotgbot.Message {
			return textMessage(privateChat(), "hello")
		}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dispatched := make(chan dispatchRecord, 8)
			env := newTestEnv(&fakePlugin{
				name:     "media",
				handlers: []plugin.Handler{commandHandler(tc.trigger, dispatched)},
			})

			env.process(t, messageUpdate(tc.message()))

			if tc.want {
				expectDispatch(t, dispatched)
			} else {
				expectNoDispatch(t, dispatched)
			}
		})
	}
}

func TestEntityTrigger(t *testing.T) {
	dispatched := make(chan dispatchRecord, 8)
	env := newTestEnv(&fakePlugin{
		name:     "urls",
		handlers: []plugin.Handler{commandHandler(tgUtils.EntityTypeURL, dispatched)},
	})

	msg := textMessage(privateChat(), "check https://example.com")
	msg.Entities = []gotgbot.MessageEntity{{Type: "url", Offset: 6, Length: 19}}
	env.process(t, messageUpdate(msg))
	expectDispatch(t, dispatched)

	env.process(t, messageUpdate(textMessage(privateChat(), "no entities here")))
	expectNoDispatch(t, dispatched)
}

func TestEntityTriggerCaption(t *testing.T) {
	dispatched := make(chan dispatchRecord, 8)
	env := newTestEnv(&fakePlugin{
		name:     "urls",
		handlers: []plugin.Handler{commandHandler(tgUtils.EntityTypeURL, dispatched)},
	})

	msg := textMessage(privateChat(), "")
	msg.Photo = []gotgbot.PhotoSize{{FileId: "p1", Width: 1, Height: 1}}
	msg.Caption = "https://example.com"
	msg.CaptionEntities = []gotgbot.MessageEntity{{Type: "url", Offset: 0, Length: 19}}
	env.process(t, messageUpdate(msg))

	expectDispatch(t, dispatched)
}

func TestHandlerErrorRepliesToUser(t *testing.T) {
	env := newTestEnv(&fakePlugin{
		name: "failing",
		handlers: []plugin.Handler{&plugin.CommandHandler{
			Trigger: regexp.MustCompile(`^/fail$`),
			HandlerFunc: func(*gotgbot.Bot, plugin.GobotContext) error {
				return errors.New("boom")
			},
		}},
	})

	env.process(t, messageUpdate(textMessage(privateChat(), "/fail")))

	r := expectRequest(t, env.client, "sendMessage")
	text, _ := r.params["text"].(string)
	if text == "" {
		t.Error("expected error reply with text")
	}
}

func TestHandlerPanicIsRecovered(t *testing.T) {
	env := newTestEnv(&fakePlugin{
		name: "panicking",
		handlers: []plugin.Handler{&plugin.CommandHandler{
			Trigger: regexp.MustCompile(`^/panic$`),
			HandlerFunc: func(*gotgbot.Bot, plugin.GobotContext) error {
				panic("oh no")
			},
		}},
	})

	env.process(t, messageUpdate(textMessage(privateChat(), "/panic")))

	expectRequest(t, env.client, "sendMessage")
}

func TestUserJoined(t *testing.T) {
	env := newTestEnv()

	msg := textMessage(groupChat(), "")
	msg.NewChatMembers = []gotgbot.User{{Id: 1}, {Id: 2}}
	env.process(t, messageUpdate(msg))

	if len(env.chatsUsers.batches) != 1 || len(env.chatsUsers.batches[0]) != 2 {
		t.Errorf("expected one batch with two users, got %v", env.chatsUsers.batches)
	}
}

func TestUserLeft(t *testing.T) {
	env := newTestEnv()

	msg := textMessage(groupChat(), "")
	msg.LeftChatMember = &gotgbot.User{Id: 55}
	env.process(t, messageUpdate(msg))

	if len(env.chatsUsers.left) != 1 || env.chatsUsers.left[0] != 55 {
		t.Errorf("expected user 55 to leave, got %v", env.chatsUsers.left)
	}
}

func TestBotLeftIsIgnored(t *testing.T) {
	env := newTestEnv()

	msg := textMessage(groupChat(), "")
	msg.LeftChatMember = &gotgbot.User{Id: 56, IsBot: true}
	env.process(t, messageUpdate(msg))

	if len(env.chatsUsers.left) != 0 {
		t.Errorf("bots leaving must be ignored, got %v", env.chatsUsers.left)
	}
}

func TestChatMetaUpdatesAreIgnored(t *testing.T) {
	dispatched := make(chan dispatchRecord, 8)
	env := newTestEnv(&fakePlugin{
		name:     "any",
		handlers: []plugin.Handler{commandHandler(tgUtils.AnyMsg, dispatched)},
	})

	msg := textMessage(groupChat(), "")
	msg.NewChatTitle = "New Title"
	env.process(t, messageUpdate(msg))

	expectNoDispatch(t, dispatched)
}

func callbackHandler(trigger *regexp.Regexp, dispatched chan dispatchRecord) *plugin.CallbackHandler {
	return &plugin.CallbackHandler{
		Trigger: trigger,
		HandlerFunc: func(_ *gotgbot.Bot, c plugin.GobotContext) error {
			dispatched <- dispatchRecord{matches: c.Matches, namedMatches: c.NamedMatches}
			return nil
		},
	}
}

func TestCallbackDispatch(t *testing.T) {
	dispatched := make(chan dispatchRecord, 8)
	env := newTestEnv(&fakePlugin{
		name:     "buttons",
		handlers: []plugin.Handler{callbackHandler(regexp.MustCompile(`^btn:(?P<action>\w+)$`), dispatched)},
	})

	env.process(t, callbackUpdate("btn:save", privateChat(), time.Now().Unix()))

	r := expectDispatch(t, dispatched)
	if r.namedMatches["action"] != "save" {
		t.Errorf("unexpected named matches: %v", r.namedMatches)
	}
}

func messagelessCallbackUpdate(data string) *gotgbot.Update {
	return &gotgbot.Update{
		UpdateId: 2,
		CallbackQuery: &gotgbot.CallbackQuery{
			Id:              "cb2",
			From:            gotgbot.User{Id: testUserID, FirstName: "Tester"},
			Data:            data,
			InlineMessageId: "inline1",
		},
	}
}

func TestCallbackWithoutMessage(t *testing.T) {
	dispatched := make(chan dispatchRecord, 8)
	handler := callbackHandler(regexp.MustCompile(`^btn$`), dispatched)
	handler.Cooldown = 10 * time.Second
	env := newTestEnv(&fakePlugin{name: "buttons", handlers: []plugin.Handler{handler}})

	env.process(t, messagelessCallbackUpdate("btn"))

	expectDispatch(t, dispatched)
}

func TestCallbackWithoutMessageHandlerError(t *testing.T) {
	done := make(chan struct{}, 2)
	env := newTestEnv(&fakePlugin{
		name: "buttons",
		handlers: []plugin.Handler{&plugin.CallbackHandler{
			Trigger: regexp.MustCompile(`^btn$`),
			HandlerFunc: func(*gotgbot.Bot, plugin.GobotContext) error {
				done <- struct{}{}
				return errors.New("boom")
			},
		}},
	})

	env.process(t, messagelessCallbackUpdate("btn"))

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("handler was not dispatched")
	}
	time.Sleep(100 * time.Millisecond)
}

func TestCallbackEmptyDataIsAnswered(t *testing.T) {
	dispatched := make(chan dispatchRecord, 8)
	env := newTestEnv(&fakePlugin{
		name:     "buttons",
		handlers: []plugin.Handler{callbackHandler(regexp.MustCompile(`.*`), dispatched)},
	})

	env.process(t, callbackUpdate("", privateChat(), time.Now().Unix()))

	expectRequest(t, env.client, "answerCallbackQuery")
	expectNoDispatch(t, dispatched)
}

func TestCallbackNotAllowed(t *testing.T) {
	dispatched := make(chan dispatchRecord, 8)
	env := newTestEnv(&fakePlugin{
		name:     "buttons",
		handlers: []plugin.Handler{callbackHandler(regexp.MustCompile(`^btn$`), dispatched)},
	})
	env.allow.userAllowed = false
	env.allow.chatAllowed = false

	env.process(t, callbackUpdate("btn", privateChat(), time.Now().Unix()))

	r := expectRequest(t, env.client, "answerCallbackQuery")
	if alert, _ := r.params["show_alert"].(bool); !alert {
		t.Error("expected alert answer for denied user")
	}
	expectNoDispatch(t, dispatched)
}

func TestCallbackPluginDisabled(t *testing.T) {
	dispatched := make(chan dispatchRecord, 8)
	env := newTestEnv(&fakePlugin{
		name:     "buttons",
		handlers: []plugin.Handler{callbackHandler(regexp.MustCompile(`^btn$`), dispatched)},
	})
	env.manager.disabledGlobally["buttons"] = true

	env.process(t, callbackUpdate("btn", privateChat(), time.Now().Unix()))

	expectRequest(t, env.client, "answerCallbackQuery")
	expectNoDispatch(t, dispatched)
}

func TestCallbackAdminOnly(t *testing.T) {
	dispatched := make(chan dispatchRecord, 8)
	handler := callbackHandler(regexp.MustCompile(`^btn$`), dispatched)
	handler.AdminOnly = true
	env := newTestEnv(&fakePlugin{name: "buttons", handlers: []plugin.Handler{handler}})

	env.process(t, callbackUpdate("btn", privateChat(), time.Now().Unix()))

	expectRequest(t, env.client, "answerCallbackQuery")
	expectNoDispatch(t, dispatched)
}

func TestCallbackCooldown(t *testing.T) {
	dispatched := make(chan dispatchRecord, 8)
	handler := callbackHandler(regexp.MustCompile(`^btn$`), dispatched)
	handler.Cooldown = 10 * time.Second
	env := newTestEnv(&fakePlugin{name: "buttons", handlers: []plugin.Handler{handler}})

	env.process(t, callbackUpdate("btn", privateChat(), time.Now().Unix()))
	expectRequest(t, env.client, "answerCallbackQuery")
	expectNoDispatch(t, dispatched)

	env.process(t, callbackUpdate("btn", privateChat(), time.Now().Add(-time.Hour).Unix()))
	expectDispatch(t, dispatched)
}

func TestCallbackDeleteButton(t *testing.T) {
	dispatched := make(chan dispatchRecord, 8)
	handler := callbackHandler(regexp.MustCompile(`^btn$`), dispatched)
	handler.DeleteButton = true
	env := newTestEnv(&fakePlugin{name: "buttons", handlers: []plugin.Handler{handler}})

	env.process(t, callbackUpdate("btn", privateChat(), time.Now().Unix()))

	expectDispatch(t, dispatched)
	expectRequest(t, env.client, "editMessageReplyMarkup")
}

func inlineHandler(trigger *regexp.Regexp, dispatched chan dispatchRecord) *plugin.InlineHandler {
	return &plugin.InlineHandler{
		Trigger:             trigger,
		CanBeUsedByEveryone: true,
		HandlerFunc: func(_ *gotgbot.Bot, c plugin.GobotContext) error {
			dispatched <- dispatchRecord{matches: c.Matches, namedMatches: c.NamedMatches}
			return nil
		},
	}
}

func TestInlineQueryDispatch(t *testing.T) {
	dispatched := make(chan dispatchRecord, 8)
	env := newTestEnv(&fakePlugin{
		name:     "search",
		handlers: []plugin.Handler{inlineHandler(regexp.MustCompile(`^find (?P<term>\w+)$`), dispatched)},
	})

	env.process(t, inlineQueryUpdate("find cats"))

	r := expectDispatch(t, dispatched)
	if r.namedMatches["term"] != "cats" {
		t.Errorf("unexpected named matches: %v", r.namedMatches)
	}
}

func TestInlineQueryEmptyIsAnswered(t *testing.T) {
	dispatched := make(chan dispatchRecord, 8)
	env := newTestEnv(&fakePlugin{
		name:     "search",
		handlers: []plugin.Handler{inlineHandler(regexp.MustCompile(`.*`), dispatched)},
	})

	env.process(t, inlineQueryUpdate(""))

	expectRequest(t, env.client, "answerInlineQuery")
	expectNoDispatch(t, dispatched)
}

func TestInlineQueryPluginDisabled(t *testing.T) {
	dispatched := make(chan dispatchRecord, 8)
	env := newTestEnv(&fakePlugin{
		name:     "search",
		handlers: []plugin.Handler{inlineHandler(regexp.MustCompile(`^find$`), dispatched)},
	})
	env.manager.disabledGlobally["search"] = true

	env.process(t, inlineQueryUpdate("find"))

	expectRequest(t, env.client, "answerInlineQuery")
	expectNoDispatch(t, dispatched)
}

func TestInlineQueryRestrictedToAllowedUsers(t *testing.T) {
	dispatched := make(chan dispatchRecord, 8)
	handler := inlineHandler(regexp.MustCompile(`^find$`), dispatched)
	handler.CanBeUsedByEveryone = false
	env := newTestEnv(&fakePlugin{name: "search", handlers: []plugin.Handler{handler}})
	env.allow.userAllowed = false

	env.process(t, inlineQueryUpdate("find"))
	expectRequest(t, env.client, "answerInlineQuery")
	expectNoDispatch(t, dispatched)

	env.allow.userAllowed = true
	env.process(t, inlineQueryUpdate("find"))
	expectDispatch(t, dispatched)
}
