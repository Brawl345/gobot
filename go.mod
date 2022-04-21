module github.com/Brawl345/gobot

// +heroku goVersion go1.18
go 1.18

require (
	github.com/go-sql-driver/mysql v1.6.0
	github.com/jmoiron/sqlx v1.3.5
	github.com/joho/godotenv v1.4.0
	github.com/rs/xid v1.4.0
	github.com/rs/zerolog v1.26.1
	github.com/rubenv/sql-migrate v1.1.1
	github.com/sosodev/duration v0.0.0-20220124054057-cb2cd96dd316
	golang.org/x/exp v0.0.0-20220407100705-7b9b53b0aca4
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	gopkg.in/guregu/null.v4 v4.0.0
	gopkg.in/telebot.v3 v3.0.0
)

require github.com/go-gorp/gorp/v3 v3.0.2 // indirect
