package analytics

var (
	AmplitudeKey = "2f88e103ec6463ae4af81f0ef6c16c09"
)

const (

	// Component
	AnalyticsComponent  Component = "analytics"
	GasStationComponent Component = "gas station"
	L1NodeComponent     Component = "l1 node"
	RollupComponent     Component = "rollup"
	OPinitComponent     Component = "opinit"
	RelayerComponent    Component = "relayer"
	UpgradeComponent    Component = "upgrade"

	// EventKeys
	ComponentEventKey  string = "component"
	FeatureEventKey    string = "feature"
	CommandEventKey    string = "command"
	OptionEventKey     string = "option"
	L1ChainIdEventKey  string = "l1_chain_id"
	ErrorEventKey      string = "panic_error"
	ModelNameKey       string = "model_name"
	GenerateKeyfileKey string = "generate_key_file"
	KeyFileKey         string = "key_file"
	WithConfigKey      string = "with_config"
	VmKey              string = "vm"
	BotTypeKey         string = "bot_type"

	// Event
	RunEvent       Event = "run"
	CompletedEvent Event = "completed"

	// Init Event
	InitActionSelected             Event = "init_action_selected"
	ExistingAppReplaceSelected     Event = "existing_app_replace_selected"
	L1NetworkSelected              Event = "l1_network_selected"
	Interrupted                    Event = "interrupted"
	Panicked                       Event = "panicked"
	L1NodeVersionSelected          Event = "l1_node_version_selected"
	PruningStrategySelected        Event = "pruning_strategy_selected"
	ExistingGenesisReplaceSelected Event = "existing_genesis_replace_selected"
	SyncMethodSelected             Event = "sync_method_selected"
	CosmovisorAutoUpgradeSelected  Event = "cosmovisor_auto_upgrade_selected"
	ExistingDataReplaceSelected    Event = "existing_data_replace_selected"
	FeaturesEnabled                Event = "feature_enabled"

	// Rollup Event
	VmTypeSelected                        Event = "vm_type_selected"
	OpBridgeBatchSubmissionTargetSelected Event = "op_bridge_batch_submission_selected"
	EnableOracleSelected                  Event = "enable_oracle_selected"
	SystemKeysSelected                    Event = "system_keys_selected"
	AccountsFundingPresetSelected         Event = "accounts_funding_preset_selected"
	AddGasStationToGenesisSelected        Event = "add_gas_station_to_genesis_selected"
	AddGenesisAccountsSelected            Event = "add_genesis_accounts_selected"

	// Opinit Event
	OPInitBotInitSelected           Event = "opinit_bot_init_selected"
	ResetDBSelected                 Event = "reset_db_selected"
	UseCurrentConfigSelected        Event = "use_current_config_selected"
	PrefillFromArtifactsSelected    Event = "prefill_from_artifacts_selected"
	L1PrefillSelected               Event = "l1_prefill_selected"
	DALayerSelected                 Event = "da_layer_selected"
	ImportKeysFromArtifactsSelected Event = "import_keys_from_artifacts_selected"
	RecoverKeySelected              Event = "recover_key_selected"

	// Relayer
	RelayerRollupSelected              Event = "relayer_rollup_selected"
	RelayerL2Selected                  Event = "relayer_l2_selected"
	SettingUpIBCChannelsMethodSelected Event = "setting_up_ibc_channels_method_selected"
	IBCChannelsSelected                Event = "ibc_channel_selected"
	UseChallengerKeySelected           Event = "use_challenger_key_selected"

	// GasStation
	GasStationMethodSelected Event = "gas_station_method_selected"
)

var (
	SetupL1NodeFeature     Feature = Feature{Name: "setup l1 node", Component: L1NodeComponent}
	RollupLaunchFeature    Feature = Feature{Name: "launch rollup", Component: RollupComponent}
	SetupOPinitBotFeature  Feature = Feature{Name: "setup opinit bot", Component: OPinitComponent}
	SetupOPinitKeysFeature Feature = Feature{Name: "setup opinit keys", Component: OPinitComponent}
	ResetOPinitBotFeature  Feature = Feature{Name: "reset opinit bot", Component: OPinitComponent}
	SetupGasStationFeature Feature = Feature{Name: "setup gas station", Component: GasStationComponent}
	SetupRelayerFeature    Feature = Feature{Name: "setup relayer", Component: RelayerComponent}
	OptOutAnalyticsFeature Feature = Feature{Name: "opt-out analytics", Component: AnalyticsComponent}
)

type Component string

type Feature struct {
	Name      string
	Component Component
}

type Event string
