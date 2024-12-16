package service

type CommandName string

const (
	UpgradeableInitia    CommandName = "upgradable_initia"
	NonUpgradeableInitia CommandName = "non_upgradable_initia"
	Minitia              CommandName = "minitia"
	OPinitExecutor       CommandName = "executor"
	OPinitChallenger     CommandName = "challenger"
	Relayer              CommandName = "relayer"
)

func (cmd CommandName) MustGetBinaryName() string {
	switch cmd {
	case UpgradeableInitia, NonUpgradeableInitia:
		return "cosmovisor"
	case Minitia:
		return "minitiad"
	case OPinitExecutor, OPinitChallenger:
		return "opinitd"
	case Relayer:
		return "hermes"
	default:
		panic("unsupported command")
	}
}

func (cmd CommandName) MustGetServiceSlug() string {
	switch cmd {
	case UpgradeableInitia:
		return "initiad"
	case NonUpgradeableInitia:
		return "initiad"
	case Minitia:
		return "minitiad"
	case OPinitExecutor:
		return "opinitd.executor"
	case OPinitChallenger:
		return "opinitd.challenger"
	case Relayer:
		return "hermes"
	default:
		panic("unsupported command")
	}
}
