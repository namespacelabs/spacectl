package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

func NewVersionCmd(version, commit, date string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version number of the Space CLI",
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		var w io.Writer = os.Stdout
		if output, _ := cmd.Flags().GetString("output"); output == "json" {
			return outputVersionJSON(w, version, commit, date)
		}

		outputVersionText(w, version, commit, date)
		return nil
	}

	return cmd
}

func outputVersionJSON(w io.Writer, version, commit, date string) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(map[string]string{
		"version": version,
		"commit":  commit,
		"date":    date,
	})
}

func outputVersionText(_ io.Writer, version, commit, date string) {
	slog.Info(fmt.Sprintf("Space CLI %s (commit: %s, built at: %s)", version, commit, date))
}
