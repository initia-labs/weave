package weaveinit

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/initia-labs/weave/analytics"
	"github.com/initia-labs/weave/common"
	weavecontext "github.com/initia-labs/weave/context"
	"github.com/initia-labs/weave/models/initia"
	"github.com/initia-labs/weave/models/minitia"
	"github.com/initia-labs/weave/models/opinit_bots"
	"github.com/initia-labs/weave/models/relayer"
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
	RunL1NodeOption       Option = "Run an L1 node"
	LaunchNewRollupOption Option = "Launch a new rollup"
	RunOPBotsOption       Option = "Run OPinit bots"
	RunRelayerOption      Option = "Run a relayer"
)

func GetWeaveInitOptions() []Option {
	options := []Option{
		RunL1NodeOption,
		LaunchNewRollupOption,
		RunOPBotsOption,
		RunRelayerOption,
	}

	return options
}

func NewWeaveInit() *WeaveInit {
	ctx := weavecontext.NewAppContext(NewWeaveInitState())
	tooltips := []ui.Tooltip{
		ui.NewTooltip(string(RunL1NodeOption), "Bootstrap an Initia Layer 1 full node to be able to join the network whether it's mainnet, testnet, or your own local network. Weave also make state-syncing and automatic upgrades super easy for you.", "", []string{}, []string{}, []string{}),
		ui.NewTooltip(string(LaunchNewRollupOption), "Customize and deploy a new rollup on Initia in less than 5 minutes. This process includes configuring your rollup components (chain ID, gas, optimistic bridge, etc.) and fund OPinit bots to facilitate communications between your rollup and the underlying Initia L1.", "", []string{}, []string{}, []string{}),
		ui.NewTooltip(string(RunOPBotsOption), "Configure and run OPinit bots, the glue between rollup and the underlying Initia L1.", "", []string{}, []string{}, []string{}),
		ui.NewTooltip(string(RunRelayerOption), "Run a relayer to facilitate communications between your rollup and the underlying Initia L1.", "", []string{}, []string{}, []string{}),
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

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return m, m.HandlePanic(fmt.Errorf("cannot get user home directory: %v", err))
	}

	selected, cmd := m.Select(msg)
	if selected != nil {
		windowWidth := weavecontext.GetWindowWidth(m.Ctx)
		switch *selected {
		case RunL1NodeOption:
			ctx := weavecontext.NewAppContext(initia.NewRunL1NodeState())
			ctx = weavecontext.SetInitiaHome(ctx, filepath.Join(homeDir, common.InitiaDirectory))
			ctx = weavecontext.SetWindowWidth(ctx, windowWidth)

			analytics.AppendGlobalEventProperties(map[string]interface{}{
				analytics.ComponentEventKey: analytics.L1NodeComponent,
				analytics.FeatureEventKey:   analytics.SetupL1NodeFeature.Name,
			})
			analytics.TrackEvent(analytics.InitActionSelected, analytics.NewEmptyEvent().Add(analytics.OptionEventKey, analytics.SetupL1NodeFeature.Name))

			model, err := initia.NewRunL1NodeNetworkSelect(ctx)
			if err != nil {
				return m, m.HandlePanic(err)
			}
			return model, nil
		case LaunchNewRollupOption:
			ctx := weavecontext.NewAppContext(*minitia.NewLaunchState())
			ctx = weavecontext.SetMinitiaHome(ctx, filepath.Join(homeDir, common.MinitiaDirectory))
			ctx = weavecontext.SetOPInitHome(ctx, filepath.Join(homeDir, common.OPinitDirectory))
			ctx = weavecontext.SetWindowWidth(ctx, windowWidth)

			analytics.AppendGlobalEventProperties(map[string]interface{}{
				analytics.ComponentEventKey: analytics.RollupComponent,
				analytics.FeatureEventKey:   analytics.RollupLaunchFeature.Name,
			})
			analytics.TrackEvent(analytics.InitActionSelected, analytics.NewEmptyEvent().Add(analytics.OptionEventKey, analytics.RollupLaunchFeature.Name))

			minitiaChecker := minitia.NewExistingMinitiaChecker(ctx)
			return minitiaChecker, minitiaChecker.Init()
		case RunOPBotsOption:
			ctx := weavecontext.NewAppContext(opinit_bots.NewOPInitBotsState())
			ctx = weavecontext.SetMinitiaHome(ctx, filepath.Join(homeDir, common.MinitiaDirectory))
			ctx = weavecontext.SetOPInitHome(ctx, filepath.Join(homeDir, common.OPinitDirectory))
			ctx = weavecontext.SetWindowWidth(ctx, windowWidth)

			analytics.AppendGlobalEventProperties(map[string]interface{}{
				analytics.ComponentEventKey: analytics.OPinitComponent,
				analytics.FeatureEventKey:   analytics.SetupOPinitBotFeature.Name,
			})
			analytics.TrackEvent(analytics.InitActionSelected, analytics.NewEmptyEvent().Add(analytics.OptionEventKey, analytics.SetupOPinitBotFeature.Name))

			model := opinit_bots.NewEnsureOPInitBotsBinaryLoadingModel(
				ctx,
				func(nextCtx context.Context) (tea.Model, error) {
					return opinit_bots.ProcessMinitiaConfig(nextCtx, opinit_bots.NewOPInitBotInitSelector)
				},
			)
			return model, model.Init()
		case RunRelayerOption:
			ctx := weavecontext.NewAppContext(relayer.NewRelayerState())
			ctx = weavecontext.SetMinitiaHome(ctx, filepath.Join(homeDir, common.MinitiaDirectory))
			ctx = weavecontext.SetWindowWidth(ctx, windowWidth)

			analytics.AppendGlobalEventProperties(map[string]interface{}{
				analytics.ComponentEventKey: analytics.RelayerComponent,
				analytics.FeatureEventKey:   analytics.SetupRelayerFeature.Name,
			})
			analytics.TrackEvent(analytics.InitActionSelected, analytics.NewEmptyEvent().Add(analytics.OptionEventKey, analytics.SetupRelayerFeature.Name))

			model, err := relayer.NewRollupSelect(ctx)
			if err != nil {
				return m, m.HandlePanic(err)
			}
			return model, nil
		}
	}

	return m, cmd
}

func (m *WeaveInit) View() string {
	m.Selector.ViewTooltip(m.Ctx)
	return m.WrapView(styles.RenderPrompt("What do you want to do?", []string{}, styles.Question) + m.Selector.View())
}
