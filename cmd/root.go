package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.chrastecky.dev/fio-client/fioclient"
	"go.chrastecky.dev/fio-client/fioclient/types"
)

var cfgFile string
var client fioclient.Client
var rootCmd = &cobra.Command{
	Use:   "fio",
	Short: "tool for managing your Fio bank account from CLI",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, fmt.Errorf("failed executing command: %w", err))
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.fio.yaml)")
	rootCmd.PersistentFlags().String("db", "", "database file location")
	rootCmd.PersistentFlags().String("encryption-password", "", "database encryption password")
	rootCmd.PersistentFlags().String("current-account", "", "the account all account operations are being done on")

	viper.BindPFlag("database", rootCmd.PersistentFlags().Lookup("db"))
	viper.BindPFlag("encryption-password", rootCmd.PersistentFlags().Lookup("encryption-password"))
	viper.BindPFlag("current-account", rootCmd.PersistentFlags().Lookup("current-account"))
}

func initConfig() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Errorf("failed getting home directory: %w", err))
		os.Exit(1)
	}

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		cfgFile = filepath.Join(home, ".fio.yaml")
		viper.AddConfigPath(home)
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName(".fio")
	}

	viper.SetEnvPrefix("FIO")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	_ = viper.ReadInConfig()

	dbPath := viper.GetString("db")
	password := viper.GetString("encryption-password")
	if dbPath == "" {
		cfgDir, err := os.UserConfigDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, fmt.Errorf("failed getting user config directory: %w", err))
			os.Exit(1)
		}
		dbPath = filepath.Join(cfgDir, "fio-cli", "fio-cli.db")
	}
	if password == "" {
		fmt.Fprintf(os.Stderr, "Please specify the password using the FIO_ENCRYPTION_PASSWORD env var.\n")
		os.Exit(1)
	}

	if _, err := os.Stat(filepath.Dir(dbPath)); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(dbPath), 0700); err != nil {
			fmt.Fprintln(os.Stderr, fmt.Errorf("failed creating fio-cli.db directory: %w", err))
			os.Exit(1)
		}
	}
	client, err = fioclient.New(
		fioclient.WithDatabase(dbPath, types.NewStringSecretKey(password)),
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Errorf("failed creating fio client: %w", err))
		os.Exit(1)
	}

	if viper.GetString("current-account") == "" {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		all, err := client.Accounts(ctx)
		if err == nil && len(all) > 0 {
			viper.Set("current-account", all[0].AccountNumber)
		}
	}
}
