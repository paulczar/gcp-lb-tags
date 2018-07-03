package util

import (
	"log"

	"github.com/spf13/cobra"
)

// GetFlagStringSlice can be used to accept multiple argument with flag repetition (e.g. -f arg1,arg2 -f arg3 ...)
func GetFlagStringSlice(cmd *cobra.Command, flag string) []string {
	s, err := cmd.Flags().GetStringSlice(flag)
	if err != nil {
		log.Fatalf("error accessing flag %s for command %s: %v", flag, cmd.Name(), err)
	}
	return s
}
