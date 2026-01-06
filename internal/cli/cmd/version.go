package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

func NewVersionCmd(version, commit, date string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version number of the Space CLI",
	}

	outputJSON := cmd.Flags().Bool("json", false, "Output result as JSON to stdout.")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		var w io.Writer = os.Stdout
		if *outputJSON {
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

func outputVersionText(w io.Writer, version, commit, date string) {
	fmt.Fprintf(w, "Space CLI %s (commit: %s, built at: %s)\n", version, commit, date)
}
