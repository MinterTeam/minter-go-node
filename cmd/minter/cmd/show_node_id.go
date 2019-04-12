package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/tendermint/tendermint/p2p"
)

var ShowNodeId = &cobra.Command{
	Use:   "show_node_id",
	Short: "Show this node's ID",
	RunE:  showNodeId,
}

func showNodeId(cmd *cobra.Command, args []string) error {
	nodeKey, err := p2p.LoadNodeKey(cfg.NodeKeyFile())
	if err != nil {
		return err
	}

	fmt.Println(nodeKey.ID())
	return nil
}
