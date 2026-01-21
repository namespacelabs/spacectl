package main

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/namespacelabs/space/internal/cli/cmd"
	"github.com/namespacelabs/space/internal/log"
)

const defaultLogLevel = "info"

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

	loglvl := cli.PersistentFlags().String("log_level", defaultLogLevel, "Log level (debug, info, warn, error)")
	outputFlag := cli.PersistentFlags().StringP("output", "o", "plain", "Output format: plain or json.")

	cli.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		logDest := io.Writer(os.Stdout)
		if *outputFlag == "json" {
			logDest = os.Stderr
		}
		return setLogger(*loglvl, logDest)
	}

	cli.AddCommand(cmd.NewCacheCmd())
	cli.AddCommand(cmd.NewVersionCmd(Version, Commit, Date))

	if err := cli.Execute(); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}

func setLogger(lvl string, w io.Writer) error {
	if strings.ToLower(os.Getenv("GITHUB_ACTIONS")) == "true" {
		return withGithubLogger(w)
	}

	return withDefaultLogger(lvl, w)
}

func withGithubLogger(w io.Writer) error {
	logger := slog.New(log.NewGithubHandler(w))
	slog.SetDefault(logger)
	return nil
}

func withDefaultLogger(lvl string, w io.Writer) error {
	slogLvl, err := parseLogLevel(lvl)
	if err != nil {
		return fmt.Errorf("invalid log level: %w", err)
	}

	logger := slog.New(log.NewPlainHandler(w, &log.PlainHandlerOptions{
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
		return slog.LevelInfo, fmt.Errorf("unknown log level %q", str)
	}
	return lvl, nil
}
