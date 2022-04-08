# Gobot

Multi-purpose bot for the Telegram Messenger based on [Telebot](https://github.com/tucnak/telebot/) and inspired
by [Python-Telegram-Bot](https://github.com/python-telegram-bot/python-telegram-bot).

**The source code is here so YOU can do what you want with it - don't ask questions about it nor ask for help how to use
it! The code might contain ugly architecture, but it solves my specific problems.**

## Features

* Written in Go
* Uses MySQL database
* Supports plugins
* Whitelist included
* Supports webhooks and long-polling

## Usage

1. Download binaries from Actions tab or build it yourself (`go build`)
2. Copy `.env.example`to `.env` and fill it in (you can also use environment variables)
3. Run it!

### Using a webhook

To use a webhook, set the webhook-related variables. If you don't, long-polling will be used.

Example with [Hookdeck](https://hookdeck.com/):

1. Copy your `https://events.hookdeck.com/e/...` URL to `WEBHOOK_URL` variable
2. Choose a webhook port (e.g. `41320`) - be careful, no error will be shown if the port is already in use (limitation
   of telebot)!
3. Use the Hookdeck CLI: `hookdeck listen 41320 [SOURCE]`

### More options

Set the env variable `PRINT_MSGS` to some value (like `true`) to print all messages the bot receives to the terminal.

Set `DEBUG` to some value (like `true`) to enable debug logs.
