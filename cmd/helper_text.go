package cmd

import "fmt"

const (
	DocsURLPrefix = "https://docs.initia.xyz/developers/developer-guides/tools/clis/weave-cli/"
)

func SubHelperText(docPath string) string {
	return fmt.Sprintf("See '%s%s' for more information about the setup process and potential issues.", DocsURLPrefix, docPath)
}

var (
	WeaveHelperText        = fmt.Sprintf("Weave is the CLI for managing Initia deployments.\n\n%s", IntroductionHelperText)
	IntroductionHelperText = SubHelperText("introduction")
	L1NodeHelperText       = SubHelperText("initia-node")
	RollupHelperText       = SubHelperText("rollup/launch")
	OPinitBotsHelperText   = SubHelperText("rollup/opinit-bots")
	RelayerHelperText      = SubHelperText("rollup/relayer")
	GasStationHelperText   = SubHelperText("gas-station")
)
