package config

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/hippopos/downloadDockerImageWithoutDockerCLI/src/pkg/log"
)

var (
	//download
	Kconfig struct {
		DownloadDir      string
		RegistryUser     string
		RegistryPassword string
	}
)

func InitConfig() {
	initGlobal()
	Kconfig.RegistryUser = viper.GetString("registry-user")
	Kconfig.RegistryPassword = viper.GetString("registry-password")
}

func initGlobal() {
	Kconfig.DownloadDir = viper.GetString("download-dir")
	if viper.GetBool("debug") {
		log.Log.SetLevel(logrus.DebugLevel)
		log.Log.SetFormatter(&logrus.TextFormatter{TimestampFormat: "2006-01-02 15:04:05", FullTimestamp: true})
	}
}
