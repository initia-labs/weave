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

	HermesHome                 = ".hermes"
	HermesKeysDirectory        = HermesHome + "/keys"
	HermesKeyFileJson          = HermesHome + "/weave.keyfile.json"
	HermesTempMnemonicFilename = "weave.mnemonic"
)
