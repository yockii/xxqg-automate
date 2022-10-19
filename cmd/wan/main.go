package main

import (
	"github.com/panjf2000/ants/v2"
	logger "github.com/sirupsen/logrus"
	"github.com/yockii/qscore/pkg/config"
	"github.com/yockii/qscore/pkg/server"

	"xxqg-automate/internal/controller/wan"
)

func init() {
	config.DefaultInstance.SetDefault("server.port", 8080)
}

func main() {
	defer ants.Release()

	wan.InitRouter()
	logger.Error(server.Start(":" + config.GetString("server.port")))
}
