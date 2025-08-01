package service

import "fmt"

type CommandName string

const (
	UpgradableInitia    CommandName = "upgradable_initia"
	NonUpgradableInitia CommandName = "non_upgradable_initia"
	Minitia             CommandName = "minitia"
	OPinitExecutor      CommandName = "executor"
	OPinitChallenger    CommandName = "challenger"
	Relayer             CommandName = "relayer"
)

func (cmd CommandName) GetPrettyName() (string, error) {
	switch cmd {
	case UpgradableInitia, NonUpgradableInitia:
		return "initia", nil
	case Minitia:
		return "rollup", nil
	case OPinitExecutor, OPinitChallenger:
		return "opinit", nil
	case Relayer:
		return "relayer", nil
	default:
		return "", fmt.Errorf("unsupported command %s", cmd)
	}
}

func (cmd CommandName) GetInitCommand() (string, error) {
	switch cmd {
	case UpgradableInitia, NonUpgradableInitia:
		return "initia init", nil
	case Minitia:
		return "rollup launch", nil
	case OPinitExecutor, OPinitChallenger:
		return "opinit init", nil
	case Relayer:
		return "relayer init", nil
	default:
		return "", fmt.Errorf("unsupported command %s", cmd)
	}
}

func (cmd CommandName) GetBinaryName() (string, error) {
	switch cmd {
	case UpgradableInitia, NonUpgradableInitia:
		return "cosmovisor", nil
	case Minitia:
		return "minitiad", nil
	case OPinitExecutor, OPinitChallenger:
		return "opinitd", nil
	// case Relayer:
	// 	return "hermes", nil
	default:
		return "", fmt.Errorf("unsupported command: %v", cmd)
	}
}

func (cmd CommandName) GetServiceSlug() (string, error) {
	switch cmd {
	case UpgradableInitia:
		return "cosmovisor", nil
	case NonUpgradableInitia:
		return "cosmovisor", nil
	case Minitia:
		return "minitiad", nil
	case OPinitExecutor:
		return "opinitd.executor", nil
	case OPinitChallenger:
		return "opinitd.challenger", nil
	// case Relayer:
	// 	return "hermes", nil
	default:
		return "", fmt.Errorf("unsupported command: %v", cmd)
	}
}
