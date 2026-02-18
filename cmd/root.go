package cmd

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string
var apiKey string
var dbPath string

var rootCmd = &cobra.Command{
	Use:   "syntrack",
	Short: "Synthetic usage tracker for API monitoring",
	Long:  `Syntrack tracks and monitors synthetic API usage across your infrastructure.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.syntrack.yaml)")
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)
		viper.AddConfigPath(home)
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName(".syntrack")
	}

	viper.SetEnvPrefix("syntrack")
	viper.AutomaticEnv()

	godotenv.Load()

	viper.SetDefault("database_path", "usage.db")
	viper.BindEnv("api_key", "SYNTHETIC_API_KEY")
	viper.BindEnv("database_path", "DATABASE_PATH")

	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}

	apiKey = viper.GetString("api_key")
	dbPath = viper.GetString("database_path")
}
