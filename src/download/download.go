package download

import (
	"github.com/hippopos/downloadDockerImageWithoutDockerCLI/src/config"
)

func Download(images []string) {
	config.InitConfig()
	downloadImage(images)
}
