package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/hippopos/downloadDockerImageWithoutDockerCLI/src/download"
)

var (
	user        string
	password    string
	downloadDir string
	images      []string
)

func newDownloadCMD() *cobra.Command {

	var command = &cobra.Command{
		Use:   "pull",
		Short: "like docker pull",
		//Long:  `All software has versions. This is Scheduler's`,
		//TraverseChildren: true,
		Run: func(cmd *cobra.Command, args []string) {
			download.Download(args)
		},
	}
	//cobra.OnInitialize(initConfig)

	command.PersistentFlags().StringVarP(&downloadDir, "download-dir", "d", ".", "path to save images")
	command.PersistentFlags().StringVarP(&user, "registry-user", "u", "", "registry login username")
	command.PersistentFlags().StringVarP(&password, "registry-password", "p", "", "registry login password")
	viper.BindPFlags(command.PersistentFlags())
	return command
}
