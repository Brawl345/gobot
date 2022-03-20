# Gobot

Multi-purpose bot for the Telegram Messenger based on [Telebot](https://github.com/tucnak/telebot/) and inspired by [Python-Telegram-Bot](https://github.com/python-telegram-bot/python-telegram-bot).

**The source code is here so YOU can do what you want with it - don't ask questions about it nor ask for help how to use it! The code might contain ugly architecture and solves my specific problems.**

## Features
* Written in Go
* Uses MySQL database
* Supports plugins
* Whitelist included

## Usage
1. Download binaries from Actions tab or build it yourself (`go build`)
2. Copy `.env.example`to `.env` and fill it in (you can also use environment variables)
3. Run it!

### More options
Set the env variable `PRINT_MSGS` to some value (like `true`) to print all messages the bot receives to the terminal.
