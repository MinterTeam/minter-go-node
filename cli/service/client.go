package service

import (
	"context"
	"fmt"
	pb "github.com/MinterTeam/minter-go-node/cli/cli_pb"
	"github.com/c-bata/go-prompt"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/empty"
	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"github.com/marcusolsson/tui-go"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"io"
	"os"
	"strings"
	"time"
)

type managerConsoleClient struct {
	cli *cli.App
}

func newManagerConsole(cli *cli.App) *managerConsoleClient {
	return &managerConsoleClient{cli: cli}
}

func (mc *managerConsoleClient) Execute(args []string) error {
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

func (mc *managerConsoleClient) Cli(ctx context.Context) error {
	completer := completer(mc.cli.Commands)
	var history []string
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			t := prompt.Input(">>> ", completer,
				prompt.OptionHistory(history),
				prompt.OptionShowCompletionAtStart(),
				prompt.OptionAddKeyBind(prompt.KeyBind{
					Key: prompt.ControlC,
					Fn: func(b *prompt.Buffer) {
						os.Exit(0)
					},
				}),
			)
			if err := mc.Execute(strings.Fields(t)); err != nil {
				if status.Code(err) == codes.Unavailable {
					return err
				}
				_, _ = fmt.Fprintln(os.Stderr, err)
			}
			history = append(history, t)
		}
	}
}

// NewCLI return
func NewCLI(socketPath string) (interface {
	Execute(args []string) error
	Cli(ctx context.Context) error
}, error) {
	cc, err := grpc.Dial("passthrough:///unix:///"+socketPath,
		grpc.WithStreamInterceptor(grpc_retry.StreamClientInterceptor()),
		grpc.WithUnaryInterceptor(grpc_retry.UnaryClientInterceptor()),
		grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	client := pb.NewManagerServiceClient(cc)

	app := cli.NewApp()
	app.CommandNotFound = func(ctx *cli.Context, cmd string) {
		fmt.Printf("No help topic for '%v'\n", cmd)
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
			Name:    "available_versions",
			Aliases: []string{"av"},
			Usage:   "display all available versions",
			Flags: []cli.Flag{
				jsonFlag,
			},
			Action: availableVersionsCMD(client),
		},
		{
			Name:    "prune_blocks",
			Aliases: []string{"pb"},
			Usage:   "delete block information",
			Flags: []cli.Flag{
				&cli.IntFlag{Name: "from", Aliases: []string{"f"}, Required: true},
				&cli.IntFlag{Name: "to", Aliases: []string{"t"}, Required: true},
				&cli.IntFlag{Name: "batch", Aliases: []string{"b"}, Required: false, Value: 250, Usage: "the number of blocks to delete in one operation"},
			},
			Action: pruneBlocksCMD(client),
		},
		{
			Name:    "status",
			Aliases: []string{"s"},
			Usage:   "display the current status of the blockchain",
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
			Usage:   "Show dashboard",
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
	return newManagerConsole(app), nil
}

func exitCMD(_ *cli.Context) error {
	os.Exit(0)
	return nil
}

func dashboardCMD(client pb.ManagerServiceClient) func(c *cli.Context) error {
	return func(c *cli.Context) error {
		ctx, cancel := context.WithCancel(c.Context)
		defer cancel()
		response, err := client.Dashboard(ctx, &empty.Empty{})
		if err != nil {
			return err
		}
		box := tui.NewVBox()
		ui, err := tui.New(tui.NewHBox(box, tui.NewSpacer()))
		if err != nil {
			return err
		}
		ui.SetKeybinding("Esc", func() { ui.Quit() })
		ui.SetKeybinding("Ctrl+C", func() { ui.Quit() })
		ui.SetKeybinding("q", func() { ui.Quit() })
		errCh := make(chan error, 2)
		uiStart := make(chan struct{})
		go func() { uiStart <- struct{}{}; errCh <- ui.Run() }()
		<-uiStart
		defer ui.Quit()
		var dashboardFunc func(recv *pb.DashboardResponse)
		for {
			select {
			case <-c.Done():
				return c.Err()
			case err := <-errCh:
				return err
			default:
				recv, err := response.Recv()
				if err == io.EOF {
					errCh <- err
					break
				}
				if err != nil {
					errCh <- err
					break
				}
				if dashboardFunc == nil {
					dashboardFunc = updateDashboard(box, recv)
				}
				dashboardFunc(recv)
				ui.Repaint()
			}
		}
	}
}

func updateDashboard(box *tui.Box, recv *pb.DashboardResponse) func(recv *pb.DashboardResponse) {
	pubKeyText := tui.NewHBox(tui.NewLabel("Validator's Pubkey: "), tui.NewLabel(recv.ValidatorPubKey), tui.NewSpacer())
	box.Append(pubKeyText)
	maxProgress := pubKeyText.SizeHint().X
	progressBox := tui.NewHBox(tui.NewEntry(), tui.NewSpacer())
	box.Append(progressBox)

	table := tui.NewTable(0, 0)
	labelNetworkSynchronizationPercent := tui.NewLabel("")
	labelNetworkSynchronizationTime := tui.NewLabel("")
	table.AppendRow(tui.NewLabel("Network Synchronization"), tui.NewHBox(labelNetworkSynchronizationPercent, labelNetworkSynchronizationTime, tui.NewSpacer()))
	labelBlockHeight := tui.NewLabel("")
	table.AppendRow(tui.NewLabel("Block Height"), labelBlockHeight)
	labelLastBlockTime := tui.NewLabel("")
	table.AppendRow(tui.NewLabel("Latest Block Time"), labelLastBlockTime)
	labelBlockProcessingTimeAvg := tui.NewLabel("")
	table.AppendRow(tui.NewLabel("Block Processing Time (avg)"), labelBlockProcessingTimeAvg)
	labelMemoryUsage := tui.NewLabel("")
	table.AppendRow(tui.NewLabel("Memory Usage"), labelMemoryUsage)
	labelPeersCount := tui.NewLabel("")
	table.AppendRow(tui.NewLabel("Peers Count"), labelPeersCount)
	labelValidatorStatus := tui.NewLabel("")
	table.AppendRow(tui.NewLabel("Validator Status: "), labelValidatorStatus)

	labelStakeName := tui.NewLabel("")
	labelVotingPowerName := tui.NewLabel("")
	labelMissedBlocksName := tui.NewLabel("")
	labelStake := tui.NewLabel("")
	labelVotingPower := tui.NewLabel("")
	labelMissedBlocks := tui.NewLabel("")

	table.AppendRow(labelStakeName, labelStake)
	table.AppendRow(labelVotingPowerName, labelVotingPower)
	table.AppendRow(labelMissedBlocksName, labelMissedBlocks)
	box.Append(tui.NewHBox(table, tui.NewSpacer()))
	box.Append(tui.NewSpacer())

	return func(recv *pb.DashboardResponse) {
		perSync := int((float64(recv.LatestHeight) / float64(recv.MaxPeerHeight)) * 100)
		labelNetworkSynchronizationPercent.SetText(fmt.Sprintf("%d%% ", perSync))
		timeLeft := ""
		ofBlocks := ""
		progressBox.Remove(0)
		progressBox.Prepend(tui.NewEntry())
		if perSync < 100 && recv.MaxPeerHeight > 0 {
			timeLeft = "Timing..."
			if recv.TimePerBlock != 0 {
				timeLeft = fmt.Sprintf("(%s left)", time.Duration((recv.MaxPeerHeight-recv.LatestHeight)*recv.TimePerBlock).Truncate(time.Second).String())
			}
			ofBlocks = fmt.Sprintf(" of %d", recv.MaxPeerHeight)
			progress := tui.NewProgress(maxProgress)
			progress.SetCurrent(int(recv.LatestHeight) * maxProgress / (int(recv.MaxPeerHeight)))
			progressBox.Remove(0)
			progressBox.Prepend(progress)
		}
		labelNetworkSynchronizationTime.SetText(timeLeft)

		labelBlockHeight.SetText(fmt.Sprintf("%d", recv.LatestHeight) + ofBlocks)
		timestamp, _ := ptypes.Timestamp(recv.Timestamp)
		labelLastBlockTime.SetText(timestamp.Format(time.RFC3339Nano) + strings.Repeat(" ", len(time.RFC3339Nano)-len(timestamp.Format(time.RFC3339Nano))))
		labelBlockProcessingTimeAvg.SetText(fmt.Sprintf("%f sec (%f sec)", time.Duration(recv.Duration).Seconds(), time.Duration(recv.AvgBlockProcessingTime).Seconds()))
		labelMemoryUsage.SetText(fmt.Sprintf("%d MB", recv.MemoryUsage/1024/1024))
		labelPeersCount.SetText(fmt.Sprintf("%d", recv.PeersCount))
		labelValidatorStatus.SetText("Not Declared")

		labelStakeName.SetText("")
		labelVotingPowerName.SetText("")
		labelMissedBlocksName.SetText("")

		labelStake.SetText("")
		labelVotingPower.SetText("")
		labelMissedBlocks.SetText("")

		if recv.ValidatorStatus != pb.DashboardResponse_NotDeclared {
			labelValidatorStatus.SetText(recv.ValidatorStatus.String())

			labelStakeName.SetText("Stake")
			labelVotingPowerName.SetText("Voting Power")
			labelStake.SetText(recv.Stake)
			labelVotingPower.SetText(fmt.Sprintf("%d", recv.VotingPower))

			if recv.ValidatorStatus != pb.DashboardResponse_Offline {
				labelMissedBlocksName.SetText("Missed Blocks")
				labelMissedBlocks.SetText(recv.MissedBlocks)
			}
		}
	}
}

func netInfoCMD(client pb.ManagerServiceClient) func(c *cli.Context) error {
	return func(c *cli.Context) error {
		response, err := client.NetInfo(c.Context, &empty.Empty{})

		if err != nil {
			return err
		}
		if c.Bool("json") {
			bb, err := protojson.Marshal(response)
			if err != nil {
				return err
			}
			fmt.Println(string(bb))
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
			bb, err := protojson.Marshal(response)
			if err != nil {
				return err
			}
			fmt.Println(string(bb))
			return nil
		}
		fmt.Println(proto.MarshalTextString(response))
		return nil
	}
}

func availableVersionsCMD(client pb.ManagerServiceClient) func(c *cli.Context) error {
	return func(c *cli.Context) error {
		response, err := client.AvailableVersions(c.Context, &empty.Empty{})
		if err != nil {
			return err
		}
		if c.Bool("json") {
			bb, err := protojson.Marshal(response)
			if err != nil {
				return err
			}
			fmt.Println(string(bb))
			return nil
		}
		fmt.Println(fmt.Sprintf("%v", response.Heights))
		return nil
	}
}

func pruneBlocksCMD(client pb.ManagerServiceClient) func(c *cli.Context) error {
	return func(c *cli.Context) error {
		ctx, cancel := context.WithCancel(c.Context)
		defer cancel()

		stream, err := client.PruneBlocks(ctx, &pb.PruneBlocksRequest{
			FromHeight: c.Int64("from"),
			ToHeight:   c.Int64("to"),
			Batch:      c.Int64("batch"),
		})
		if err != nil {
			return err
		}

		now := time.Now()
		errCh := make(chan error)
		recvCh := make(chan *pb.PruneBlocksResponse)

		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				default:
					recv, err := stream.Recv()
					if err == io.EOF {
						close(errCh)
						return
					}
					if err != nil {
						errCh <- err
						return
					}
					recvCh <- recv
				}
			}
		}()

		for {
			select {
			case <-c.Done():
				return c.Err()
			case err, more := <-errCh:
				_ = stream.CloseSend()
				if more {
					close(errCh)
					return err
				}
				fmt.Println("OK", time.Since(now).String())
				return nil
			case recv := <-recvCh:
				var percent int64
				if recv.Total != 0 {
					percent = int64(float64(recv.Current) / float64(recv.Total) * 100.0)
				}
				fmt.Printf("%d%% successfully removed (%d of %d) %s\n", percent, recv.Current, recv.Total, time.Since(now).String())
			}
		}
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
