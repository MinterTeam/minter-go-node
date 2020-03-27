package service

import (
	"bytes"
	"context"
	"fmt"
	"github.com/MinterTeam/minter-go-node/cli/pb"
	"github.com/c-bata/go-prompt"
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/empty"
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

		if err := ui.Init(); err != nil {
			return err
		}
		defer ui.Close()

		for {
			select {
			case <-c.Done():
				return c.Err()
			case <-time.After(time.Second):
				recv, err := response.Recv()
				if err == io.EOF {
					return nil
				}
				if err != nil {
					return err
				}
				ui.Render(recvToDashboard(recv)()...)
			}
		}
	}
}

func recvToDashboard(recv *pb.DashboardResponse) func() []ui.Drawable {
	pubKeyText := widgets.NewParagraph()
	pubKeyText.Title = "Public Key Validator"
	pubKeyText.SetRect(0, 0, 110, 3)

	gauge := widgets.NewGauge()
	gauge.Title = "Network synchronization"
	gauge.SetRect(0, 3, 40, 6)

	table1 := widgets.NewTable()
	table1.Rows = [][]string{
		{"Height (HeaderTimestamp)", ""},
		{"Duration block", ""},
		{"Memory", ""},
		{"Count peers", ""},
		{"Average block processing speed", ""},
	}
	table1.SetRect(40, 3, 110, 14)

	return func() (items []ui.Drawable) {
		gauge.Percent = int((float64(recv.CurrentHeight) / float64(recv.LastHeight)) * 100)
		gauge.Title += fmt.Sprintf(" (%d%%)", gauge.Percent)
		gauge.Label = fmt.Sprintf("%s", time.Duration(float32(recv.LastHeight-recv.CurrentHeight)*recv.AverageTimeBlock).Truncate(time.Second).String())
		pubKeyText.Text = recv.PubKey

		timestamp, _ := ptypes.Timestamp(recv.Timestamp)
		table1.Rows[0][1] = fmt.Sprintf("%d (%s)", recv.CurrentHeight, timestamp.Format(time.RFC3339Nano))
		table1.Rows[1][1] = fmt.Sprintf("%f sec", time.Duration(recv.Duration).Seconds())
		table1.Rows[2][1] = fmt.Sprintf("%d MB", recv.Memory/1024/1024)
		table1.Rows[3][1] = fmt.Sprintf("%d", recv.CountPeers)
		table1.Rows[4][1] = fmt.Sprintf("%f sec", time.Duration(recv.AverageTimeBlock).Seconds())
		return append(items, gauge, pubKeyText, table1)
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
