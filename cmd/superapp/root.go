package main

import (
    "fmt"
    "os"

    "github.com/spf13/cobra"
    "github.com/spf13/viper"
)

var (
    cfgFile string
)

var rootCmd = &cobra.Command{
    Use:   "superapp",
    Short: "Super-App CLI",
    Long:  "Operational CLI for the Super-App ecosystem",
}

func init() {
    // Global flags
    rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "./internal/gateway/config/config.yaml", "path to config.yaml")
    _ = viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))

    // Env support (prefix SUPERAPP_)
    viper.SetEnvPrefix("SUPERAPP")
    viper.AutomaticEnv()
}

func Execute() {
    if err := rootCmd.Execute(); err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
}