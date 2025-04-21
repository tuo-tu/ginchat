package main

import (
	"ginchat/models"
	"ginchat/router"
	"ginchat/utils"
	"github.com/spf13/viper"
	"time"
)

func main() {
	utils.InitConfig()
	utils.InitMySQL()
	utils.InitRedis()
	InitTimer()
	r := router.Router()
	r.Run(viper.GetString("port.server"))
}

func InitTimer() {
	utils.Timer(
		time.Duration(viper.GetInt("timeout.DelayHeartbeat"))*time.Second,
		time.Duration(viper.GetInt("timeout.HeartbeatHz"))*time.Second,
		models.CleanConnection, "")
}
