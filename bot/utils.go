package bot

import (
	"gopkg.in/telebot.v3"
	"os"
	"strconv"
)

func isAdmin(user *telebot.User) bool {
	adminId, _ := strconv.ParseInt(os.Getenv("ADMIN_ID"), 10, 64)
	return adminId == user.ID
}
