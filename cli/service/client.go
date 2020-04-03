package service

import (
	"bytes"
	"context"
	"fmt"
	"github.com/MinterTeam/minter-go-node/cli/pb"
	"github.com/c-bata/go-prompt"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/marcusolsson/tui-go"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
	"io"
	"os"
	"strings"
	"time"
)

type ManagerConsole struct {
	cli *cli.App
}

func NewManagerConsole(cli *cli.App) *ManagerConsole {
	return &ManagerConsole{cli: cli}
}

func (mc *ManagerConsole) Execute(args []string) error {
	return mc.cli.Run(append(make([]string, 1, len(args)+1), args...))
}

func completer(commands cli.Commands) prompt.Completer {
	cmdHints := make([]prompt.Suggest, 0, len(commands))
	for _, command := range commands {
		cmdHints = append(cmdHints, prompt.Suggest{Text: command.Name, Description: command.Usage})
	}
	return func(doc prompt.Document) []prompt.Suggest {
		before := doc.TextBeforeCursor()
		wordsBefore := strings.Split(before, " ")
		// the command being entered is the text until the first space
		commandBefore := wordsBefore[0]
		if len(wordsBefore) == 1 {
			return prompt.FilterHasPrefix(cmdHints, commandBefore, true)
		}

		var flagHints []prompt.Suggest

		if strings.Contains(before, "--help") {
			return flagHints
		}

		for _, command := range commands {
			if !command.HasName(commandBefore) {
				continue
			}

			for _, flag := range command.VisibleFlags() {
				tag := "--" + flag.Names()[0]
				if strings.Contains(before, tag) {
					continue
				}
				if len(wordsBefore) > 2 && tag == "--help" {
					continue
				}
				neededValue := "="
				if _, ok := flag.(*cli.BoolFlag); ok {
					neededValue = " "
				}
				flagHints = append(flagHints, prompt.Suggest{
					Text:        tag + neededValue,
					Description: strings.ReplaceAll(flag.String(), "\t", " "),
				})
			}
			break
		}

		return prompt.FilterFuzzy(flagHints, wordsBefore[len(wordsBefore)-1], true)
	}
}

func (mc *ManagerConsole) Cli(ctx context.Context) {
	completer := completer(mc.cli.Commands)
	var history []string
	for {
		select {
		case <-ctx.Done():
			return
		default:
			t := prompt.Input(">>> ", completer,
				prompt.OptionHistory(history),
				prompt.OptionShowCompletionAtStart(),
			)
			if err := mc.Execute(strings.Fields(t)); err != nil {
				_, _ = fmt.Fprintln(os.Stderr, err)
			}
			history = append(history, t)
		}
	}
}

func ConfigureManagerConsole(socketPath string) (*ManagerConsole, error) {
	cc, err := grpc.Dial("passthrough:///unix:///"+socketPath, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	client := pb.NewManagerServiceClient(cc)

	app := cli.NewApp()
	app.CommandNotFound = func(ctx *cli.Context, cmd string) {
		fmt.Println(fmt.Sprintf("No help topic for '%v'", cmd))
	}
	app.UseShortOptionHandling = true
	jsonFlag := &cli.BoolFlag{Name: "json", Aliases: []string{"j"}, Required: false, Usage: "echo in json format"}

	app.Commands = []*cli.Command{
		{
			Name:    "dial_peer",
			Aliases: []string{"dp"},
			Usage:   "connect a new peer",
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "address", Aliases: []string{"a"}, Required: true, Usage: "id@ip:port"},
				&cli.BoolFlag{Name: "persistent", Aliases: []string{"p"}, Required: false},
			},
			Action: dealPeerCMD(client),
		},
		{
			Name:    "prune_blocks",
			Aliases: []string{"pb"},
			Usage:   "delete block information",
			Flags: []cli.Flag{
				&cli.IntFlag{Name: "from", Aliases: []string{"f"}, Required: true},
				&cli.IntFlag{Name: "to", Aliases: []string{"t"}, Required: true},
			},
			Action: pruneBlocksCMD(client),
		},
		{
			Name:    "status",
			Aliases: []string{"s"},
			Usage:   "display the current statusCMD of the blockchain",
			Flags: []cli.Flag{
				jsonFlag,
			},
			Action: statusCMD(client),
		},
		{
			Name:    "net_info",
			Aliases: []string{"ni"},
			Usage:   "display network data",
			Flags: []cli.Flag{
				jsonFlag,
			},
			Action: netInfoCMD(client),
		},
		{
			Name:    "dashboard",
			Aliases: []string{"db"},
			Usage:   "Show dashboard", //todo
			Action:  dashboardCMD(client),
		},
		{
			Name:    "exit",
			Aliases: []string{"e"},
			Usage:   "exit",
			Action:  exitCMD,
		},
	}

	for _, command := range app.Commands {
		command.Flags = append(command.Flags, cli.HelpFlag)
	}

	app.Setup()
	return NewManagerConsole(app), nil
}

func exitCMD(_ *cli.Context) error {
	os.Exit(0)
	return nil
}

func dashboardCMD(client pb.ManagerServiceClient) func(c *cli.Context) error {
	return func(c *cli.Context) error {
		ctx, cancel := context.WithCancel(c.Context)
		response, err := client.Dashboard(ctx, &empty.Empty{})
		if err != nil {
			return err
		}

		defer cancel()

		box := tui.NewVBox()
		ui, err := tui.New(tui.NewHBox(box, tui.NewSpacer()))
		if err != nil {
			return err
		}
		ui.SetKeybinding("Esc", func() { ui.Quit() })
		ui.SetKeybinding("q", func() { ui.Quit() })
		errCh := make(chan error)

		recv, err := response.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		dashboard := updateDashboard(box, recv)

		go func() { errCh <- ui.Run() }()

		for {
			select {
			case <-c.Done():
				return c.Err()
			case err := <-errCh:
				return err
			case <-time.After(time.Second):
				recv, err := response.Recv()
				if err == io.EOF {
					return nil
				}
				if err != nil {
					return err
				}
				dashboard(recv)
				ui.Repaint()
			}
		}
	}
}

func updateDashboard(box *tui.Box, recv *pb.DashboardResponse) func(recv *pb.DashboardResponse) {
	pubKeyText := tui.NewHBox(tui.NewLabel("Validator's Pubkey: "), tui.NewLabel(recv.ValidatorPubKey), tui.NewSpacer())
	box.Append(pubKeyText)

	progress := tui.NewProgress(int(recv.MaxPeerHeight))
	box.Append(tui.NewHBox(progress, tui.NewSpacer()))

	table := tui.NewTable(0, 0)
	labelNetworkSynchronizationPercent := tui.NewLabel(fmt.Sprintf("%d%% ", int((float64(recv.LatestHeight)/float64(recv.MaxPeerHeight))*100)))
	labelNetworkSynchronizationTime := tui.NewLabel(fmt.Sprintf("(%s left)", time.Duration((recv.MaxPeerHeight-recv.LatestHeight)*recv.TimePerBlock).Truncate(time.Second).String()))
	table.AppendRow(tui.NewLabel("Network Synchronization"), tui.NewHBox(labelNetworkSynchronizationPercent, labelNetworkSynchronizationTime, tui.NewSpacer()))
	labelBlockHeight := tui.NewLabel(fmt.Sprintf("%d of %d", recv.LatestHeight, recv.MaxPeerHeight))
	table.AppendRow(tui.NewLabel("Block Height"), labelBlockHeight)
	timestamp, _ := ptypes.Timestamp(recv.Timestamp)
	labelLastBlockTime := tui.NewLabel(timestamp.Format(time.RFC3339Nano) + strings.Repeat(" ", len(time.RFC3339Nano)-len(timestamp.Format(time.RFC3339Nano))))
	table.AppendRow(tui.NewLabel("Latest Block Time"), labelLastBlockTime)
	labelBlockProcessingTimeAvg := tui.NewLabel(fmt.Sprintf("%f sec (%f sec)", time.Duration(recv.Duration).Seconds(), time.Duration(recv.AvgBlockProcessingTime).Seconds()))
	table.AppendRow(tui.NewLabel("Block Processing Time (avg)"), labelBlockProcessingTimeAvg)
	labelMemoryUsage := tui.NewLabel(fmt.Sprintf("%d MB", recv.MemoryUsage/1024/1024))
	table.AppendRow(tui.NewLabel("Memory Usage"), labelMemoryUsage)
	labelPeersCount := tui.NewLabel(fmt.Sprintf("%d", recv.PeersCount))
	table.AppendRow(tui.NewLabel("Peers Count"), labelPeersCount)
	labelValidatorStatus := tui.NewLabel(recv.ValidatorStatus.String())
	table.AppendRow(tui.NewLabel("Validator Status: "), labelValidatorStatus)

	labelStakeName := tui.NewLabel("")
	labelVotingPowerName := tui.NewLabel("")
	labelMissedBlocksName := tui.NewLabel("")
	labelStake := tui.NewLabel("")
	labelVotingPower := tui.NewLabel("")
	labelMissedBlocks := tui.NewLabel("")
	if recv.ValidatorStatus != pb.DashboardResponse_NotDeclared {
		labelStakeName.SetText("Stake")
		labelVotingPowerName.SetText("Voting Power")
		labelMissedBlocksName.SetText("Missed Blocks")
		labelStake.SetText(recv.Stake)
		labelVotingPower.SetText(fmt.Sprintf("%d", recv.VotingPower))
		labelMissedBlocks.SetText(recv.MissedBlocks)
	}
	table.AppendRow(labelStakeName, labelStake)
	table.AppendRow(labelVotingPowerName, labelVotingPower)
	table.AppendRow(labelMissedBlocksName, labelMissedBlocks)
	box.Append(tui.NewHBox(table, tui.NewSpacer()))
	box.Append(tui.NewSpacer())

	return func(recv *pb.DashboardResponse) {
		labelNetworkSynchronizationPercent.SetText(fmt.Sprintf("%d%% ", int((float64(recv.LatestHeight)/float64(recv.MaxPeerHeight))*100)))
		timeLeft := "Timing..."
		if recv.TimePerBlock != 0 {
			timeLeft = fmt.Sprintf("(%s left)", time.Duration((recv.MaxPeerHeight-recv.LatestHeight)*recv.TimePerBlock).Truncate(time.Second).String())
		}
		labelNetworkSynchronizationTime.SetText(timeLeft)
		labelBlockHeight.SetText(fmt.Sprintf("%d of %d", recv.LatestHeight, recv.MaxPeerHeight))
		timestamp, _ := ptypes.Timestamp(recv.Timestamp)
		labelLastBlockTime.SetText(timestamp.Format(time.RFC3339Nano) + strings.Repeat(" ", len(time.RFC3339Nano)-len(timestamp.Format(time.RFC3339Nano))))
		labelBlockProcessingTimeAvg.SetText(fmt.Sprintf("%f sec (%f sec)", time.Duration(recv.Duration).Seconds(), time.Duration(recv.AvgBlockProcessingTime).Seconds()))
		labelMemoryUsage.SetText(fmt.Sprintf("%d MB", recv.MemoryUsage/1024/1024))
		labelPeersCount.SetText(fmt.Sprintf("%d", recv.PeersCount))
		labelValidatorStatus.SetText(recv.ValidatorStatus.String())
		if recv.ValidatorStatus != pb.DashboardResponse_NotDeclared {
			labelStakeName.SetText("Stake")
			labelVotingPowerName.SetText("Voting Power")
			labelMissedBlocksName.SetText("Missed Blocks")
			labelStake.SetText(recv.Stake)
			labelVotingPower.SetText(fmt.Sprintf("%d", recv.VotingPower))
			labelMissedBlocks.SetText(recv.MissedBlocks)
		}
		progress.SetMax(int(recv.MaxPeerHeight))
		progress.SetCurrent(int(recv.LatestHeight))
	}
}

func netInfoCMD(client pb.ManagerServiceClient) func(c *cli.Context) error {
	return func(c *cli.Context) error {
		response, err := client.NetInfo(c.Context, &empty.Empty{})

		if err != nil {
			return err
		}
		if c.Bool("json") {
			bb := new(bytes.Buffer)
			err := (&jsonpb.Marshaler{EmitDefaults: true}).Marshal(bb, response)
			if err != nil {
				return err
			}
			fmt.Println(string(bb.Bytes()))
			return nil
		}
		fmt.Println(proto.MarshalTextString(response))
		return nil
	}
}

func statusCMD(client pb.ManagerServiceClient) func(c *cli.Context) error {
	return func(c *cli.Context) error {
		response, err := client.Status(c.Context, &empty.Empty{})
		if err != nil {
			return err
		}
		if c.Bool("json") {
			bb := new(bytes.Buffer)
			err := (&jsonpb.Marshaler{EmitDefaults: true}).Marshal(bb, response)
			if err != nil {
				return err
			}
			fmt.Println(string(bb.Bytes()))
			return nil
		}
		fmt.Println(proto.MarshalTextString(response))
		return nil
	}
}

func pruneBlocksCMD(client pb.ManagerServiceClient) func(c *cli.Context) error {
	return func(c *cli.Context) error {
		_, err := client.PruneBlocks(c.Context, &pb.PruneBlocksRequest{
			FromHeight: c.Int64("from"),
			ToHeight:   c.Int64("to"),
		})
		if err != nil {
			return err
		}
		fmt.Println("OK")
		return nil
	}
}

func dealPeerCMD(client pb.ManagerServiceClient) func(c *cli.Context) error {
	return func(c *cli.Context) error {
		_, err := client.DealPeer(c.Context, &pb.DealPeerRequest{
			Address:    c.String("address"),
			Persistent: c.Bool("persistent"),
		})
		if err != nil {
			return err
		}
		fmt.Println("OK")
		return nil
	}
}
