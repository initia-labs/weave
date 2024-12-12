package weaveinit

import (
	tea "github.com/charmbracelet/bubbletea"

	weavecontext "github.com/initia-labs/weave/context"
	"github.com/initia-labs/weave/flags"
	"github.com/initia-labs/weave/models/initia"
	"github.com/initia-labs/weave/models/minitia"
	"github.com/initia-labs/weave/models/opinit_bots"
	"github.com/initia-labs/weave/styles"
	"github.com/initia-labs/weave/types"
	"github.com/initia-labs/weave/ui"
)

type State struct {
	weave types.WeaveState
}

func NewWeaveInitState() State {
	return State{
		weave: types.NewWeaveState(),
	}
}

func (e State) Clone() State {
	return State{
		weave: e.weave.Clone(),
	}
}

type WeaveInit struct {
	ui.Selector[Option]
	weavecontext.BaseModel
}

type Option string

const (
	RunL1NodeOption        Option = "Run L1 Node"
	LaunchNewMinitiaOption Option = "Launch New Minitia"
	SetupOPBotsKeys        Option = "Setup OPInit Bots Keys"
	InitializeOPBotsOption Option = "Initialize OPInit Bots"
	StartRelayerOption     Option = "Start a Relayer"
)

func GetWeaveInitOptions() []Option {
	options := []Option{
		RunL1NodeOption,
	}

	if flags.IsEnabled(flags.MinitiaLaunch) {
		options = append(options, LaunchNewMinitiaOption)
	}

	if flags.IsEnabled(flags.OPInitBots) {
		options = append(options, SetupOPBotsKeys)
		options = append(options, InitializeOPBotsOption)
	}

	if flags.IsEnabled(flags.Relayer) {
		options = append(options, StartRelayerOption)
	}

	return options
}

func NewWeaveInit() *WeaveInit {
	ctx := weavecontext.NewAppContext(NewWeaveInitState())
	tooltips := []ui.Tooltip{
		ui.NewTooltip(string(RunL1NodeOption), "Bootstrap an Initia Layer 1 full node to be able to join the network whether it's Mainnet, Testnet, or your own local network. Weave also make state-syncing super easy for you.", "", []string{}, []string{}, []string{}),
		ui.NewTooltip(string(LaunchNewMinitiaOption), "Customize and deploy a new Minitia, an L2 rollup on Initia in less than 5 minutes. This process includes configuring your L2 components (chain-id, gas, optimistic bridge, etc.) and fund OPinit Bots to facilitate communications between your Minitia and the underlying Initia L1.", "", []string{}, []string{}, []string{}),
		ui.NewTooltip(string(SetupOPBotsKeys), "TBD", "", []string{}, []string{}, []string{}),
		ui.NewTooltip(string(InitializeOPBotsOption), "Configure and run OPinit Bots, the glue between Minitia and the underlying Initia L1.", "", []string{}, []string{}, []string{}),
	}

	return &WeaveInit{
		BaseModel: weavecontext.BaseModel{Ctx: ctx, CannotBack: true},
		Selector: ui.Selector[Option]{
			Options:    GetWeaveInitOptions(),
			Cursor:     0,
			CannotBack: true,
			Tooltips:   &tooltips,
		},
	}
}

func (m *WeaveInit) Init() tea.Cmd {
	return nil
}

func (m *WeaveInit) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := weavecontext.HandleCommonCommands[State](m, msg); handled {
		return model, cmd
	}

	selected, cmd := m.Select(msg)
	if selected != nil {
		switch *selected {
		case RunL1NodeOption:
			ctx := weavecontext.NewAppContext(initia.RunL1NodeState{})
			return initia.NewRunL1NodeNetworkSelect(ctx), nil
		case LaunchNewMinitiaOption:
			minitiaChecker := minitia.NewExistingMinitiaChecker(weavecontext.NewAppContext(*minitia.NewLaunchState()))
			return minitiaChecker, minitiaChecker.Init()
		case SetupOPBotsKeys:
			ctx := weavecontext.NewAppContext(opinit_bots.NewOPInitBotsState())
			return opinit_bots.PrepareSetup(ctx), nil
		case InitializeOPBotsOption:
			return opinit_bots.NewOPInitBotInitSelector(weavecontext.NewAppContext(opinit_bots.NewOPInitBotsState())), nil
		}
	}

	return m, cmd
}

func (m *WeaveInit) View() string {
	m.Selector.ToggleTooltip = weavecontext.GetTooltip(m.Ctx)
	return styles.RenderPrompt("What action would you like to perform?", []string{}, styles.Question) + m.Selector.View()
}
