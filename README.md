# Gobot

[![Build](https://github.com/Brawl345/gobot/actions/workflows/build.yml/badge.svg "GitHub Actions Build Badge")](https://github.com/Brawl345/gobot/actions/workflows/build.yml) [![Deploy to Koyeb](https://www.koyeb.com/static/images/deploy/button.svg)](https://app.koyeb.com/deploy?type=git&name=gobot&ports=8080;http;/&repository=github.com/Brawl345/gobot&branch=master)



Multi-purpose bot for the Telegram Messenger based on [gotgbot](https://github.com/PaulSonOfLars/gotgbot) and inspired
by [Python-Telegram-Bot](https://github.com/python-telegram-bot/python-telegram-bot).

**The source code is here so YOU can do what you want with it - don't ask questions about it nor ask for help on how to use
it!**

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

1. Copy your `https://events.hookdeck.com/e/...` URL to `WEBHOOK_PUBLIC_URL` variable
2. Set a webhook port (e.g. `41320`) to `PORT`
3. Set `WEBHOOK_URL_PATH` to a custom path
   1. This is where the internal webhook server will listen on. Gotgbot **does not support an empty path** sadly.
   2. For Hookdeck, set "Destionation Type" to "CLI" and insert your path
4. Use the Hookdeck CLI: `hookdeck listen 41320 [SOURCE]`

### More options

Set the following variables to any value (like "`1`") to enable them:

* `PRINT_MSGS`: Print all messages the bot receives to the terminal
* `PRETTY_PRINT_LOG`: Pretty print log
* `DEBUG`: Enable debug logs (verbose, contains secrets!)
* `IGNORE_SQL_MIGRATION`: Ignore the SQL migration feature when you want to migrate yourself (for example with
  PlanetScale since it doesn't support foreign key references).
