package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/namespacelabs/space/internal/cli/cmd"
)

const defaultLogLevel = "warn"

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

func main() {
	cli := &cobra.Command{
		Use:   "space",
		Short: "CLI used for powering various Namespace functionality",
		Long:  `A CLI tool for powering various Namespace functionality.`,
	}

	loglvl := cli.PersistentFlags().String("log-level", defaultLogLevel, "Log level (debug, info, warn, error)")

	cli.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		return setLogger(*loglvl)
	}

	cli.AddCommand(cmd.NewCacheCmd())
	cli.AddCommand(cmd.NewVersionCmd(Version, Commit, Date))

	if err := cli.Execute(); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}

func setLogger(lvl string) error {
	slogLvl, err := parseLogLevel(lvl)
	if err != nil {
		return fmt.Errorf("invalid log level: %w", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slogLvl,
	}))
	slog.SetDefault(logger)
	return nil
}

func parseLogLevel(str string) (slog.Level, error) {
	if str == "" {
		str = "info"
		if envStr := os.Getenv("LOG_LEVEL"); envStr != "" {
			str = envStr
		}
	}

	var lvl slog.Level
	if err := lvl.UnmarshalText([]byte(str)); err != nil {
		return slog.LevelInfo, fmt.Errorf("unknown log level `%s`", str)
	}
	return lvl, nil
}
