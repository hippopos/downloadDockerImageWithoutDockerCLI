package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var log bool
var iFile string

var rootCmd = &cobra.Command{
	Use:     fmt.Sprintf("%s [subcommand]", os.Args[0]),
	Short:   "装机小能手",
	Long:    `koictl装机小能手: 可以快速安装 kubernetes和mongo(PSA) 集群,以及surpass opsapps`,
	Version: "0.1",
	// Run: func(cmd *cobra.Command, args []string) {
	//      v, _ := cmd.PersistentFlags().GetBool("version")
	//      fmt.Println(v, args)
	//      // Do Stuff Here
	// },

}

// Execute
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	// rootCmd.PersistentFlags().BoolVarP(&version, "version", "v", false, "Print the version number of scheduler")
	rootCmd.PersistentFlags().BoolVar(&log, "debug", false, "show debug logs")
	viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))

	cobra.OnInitialize()
	rootCmd.AddCommand(newDownloadCMD())
}
