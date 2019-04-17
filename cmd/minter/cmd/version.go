package cmd

import (
	"fmt"
	"github.com/MinterTeam/minter-go-node/version"
	"github.com/spf13/cobra"
)

var Version = &cobra.Command{
	Use:   "version",
	Short: "Show this node's version",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println(version.Version)
		return nil
	},
}
