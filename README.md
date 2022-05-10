# Gobot

[![Build](https://github.com/Brawl345/gobot/actions/workflows/build.yml/badge.svg "GitHub Actions Build Badge")](https://github.com/Brawl345/gobot/actions/workflows/build.yml) [![Deploy](https://www.herokucdn.com/deploy/button.svg)](https://heroku.com/deploy) [![Deploy to Koyeb](https://www.koyeb.com/static/images/deploy/button.svg)](https://app.koyeb.com/deploy?type=git&name=gobot&ports=8080;http;/&repository=github.com/Brawl345/gobot&branch=master)



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

1. Copy your `https://events.hookdeck.com/e/...` URL to `WEBHOOK_PUBLIC_URL` variable
2. Choose a webhook port (e.g. `41320`) - be careful, no error will be shown if the port is already in use (limitation
   of telebot)!
3. Use the Hookdeck CLI: `hookdeck listen 41320 [SOURCE]`

### More options

Set the following variables to a truthy value (like "`true`") to enable them:

* `PRINT_MSGS`: Print all messages the bot receives to the terminal
* `PRETTY_PRINT_LOG`: Pretty print log
* `DEBUG`: Enable debug logs (verbose!)
* `IGNORE_SQL_MIGRATION`: Ignore the SQL migration feature when you want to migrate yourself (for example with
  PlanetScale since it doesn't support foreign key references).
