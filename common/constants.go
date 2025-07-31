package common

const (
	WeaveDirectory     = ".weave"
	WeaveConfigFile    = WeaveDirectory + "/config.json"
	WeaveDataDirectory = WeaveDirectory + "/data"
	WeaveLogDirectory  = WeaveDirectory + "/log"

	SnapshotFilename = "snapshot.weave"

	InitiaDirectory       = ".initia"
	InitiaConfigDirectory = "/config"
	InitiaDataDirectory   = "/data"

	WeaveGasStationKeyName = "weave.GasStation"

	MinitiaDirectory           = ".minitia"
	MinitiaConfigPath          = ".minitia/config"
	MinitiaArtifactsConfigJson = "/artifacts/config.json"
	MinitiaArtifactsJson       = "/artifacts/artifacts.json"

	OPinitDirectory            = ".opinit"
	OPinitAppName              = "opinitd"
	OPinitKeyFileJson          = "/weave.keyfile.json"
	OpinitGeneratedKeyFilename = "weave.opinit.generated"

	RelayerHome                 = ".relayer"
	RelayerKeysDirectory        = RelayerHome + "/keys"
	RelayerKeyFileJson          = RelayerHome + "/weave.keyfile.json"
	RelayerTempMnemonicFilename = "weave.mnemonic"
)
