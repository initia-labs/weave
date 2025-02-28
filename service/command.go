package service

import (
	"fmt"
	"strings"
)

type Command struct {
	Name             string
	DefaultImageURL  string
	StartCommandArgs []string
	StartPortArgs    []string
	StartEnvArgs     []string
	InitCommand      string
}

const (
	INITIA_GHCR_BASE_URL = "ghcr.io/initia-labs"
)

var (
	COSMOVISOR_DOCKER_IMAGE_URL = "ghcr.io/initia-labs/cosmovisor:v1.1.1"
	OPINIT_DOCKER_IMAGE_URL     = "ghcr.io/initia-labs/opinitd:v0.1.14-2"
	HERMES_DOCKER_IMAGE_URL     = "ghcr.io/initia-labs/hermes:v1.1.1"
)

func GetMinitiaImage(vm string, version string) string {
	return fmt.Sprintf("%s/mini%s:%s", INITIA_GHCR_BASE_URL, strings.ToLower(vm), version)
}

var (
	UpgradableInitia Command = Command{
		Name:             "initia",
		DefaultImageURL:  COSMOVISOR_DOCKER_IMAGE_URL,
		StartCommandArgs: []string{"run", "start"},
		StartPortArgs: []string{
			"-p", "26656:26656",
			"-p", "26657:26657",
			"-p", "1317:1317",
			"-p", "9090:9090",
		},
		StartEnvArgs: []string{
			"-e", "LD_LIBRARY_PATH=/app/data/cosmovisor/dyld_lib",
			"-e", "DAEMON_NAME=initiad",
			"-e", "DAEMON_HOME=/app/data",
			"-e", "DAEMON_ALLOW_DOWNLOAD_BINARIES=true",
			"-e", "DAEMON_RESTART_AFTER_UPGRADE=true",
		},
		InitCommand: "initia init",
	}
	NonUpgradableInitia Command = Command{
		Name:             "initia",
		DefaultImageURL:  COSMOVISOR_DOCKER_IMAGE_URL,
		StartCommandArgs: []string{"run", "start"},
		StartPortArgs: []string{
			"-p", "26656:26656",
			"-p", "26657:26657",
			"-p", "1317:1317",
			"-p", "9090:9090",
		},
		InitCommand: "initia init",
	}
	Rollup Command = Command{
		Name:             "rollup",
		DefaultImageURL:  "",
		StartCommandArgs: []string{"start", "--home", "/app/data"},
		StartPortArgs: []string{
			"-p", "26656:26656",
			"-p", "26657:26657",
			"-p", "1317:1317",
			"-p", "9090:9090",
			"-p", "8545:8545", // JSON-RPC
			"-p", "8546:8546", // JSON-RPC-WS
		},
		InitCommand: "rollup launch",
	}
	OPinitExecutor Command = Command{
		Name:             "executor",
		DefaultImageURL:  OPINIT_DOCKER_IMAGE_URL,
		StartCommandArgs: []string{"start", "executor", "--home", "/app/data"},
		StartPortArgs: []string{
			"-p", "3000:3000",
		},
		InitCommand: "opinit init",
	}
	OPinitChallenger Command = Command{
		Name:             "challenger",
		DefaultImageURL:  OPINIT_DOCKER_IMAGE_URL,
		StartCommandArgs: []string{"start", "challenger", "--home", "/app/data"},
		StartPortArgs: []string{
			"-p", "3001:3001",
		},
		InitCommand: "opinit init",
	}
	Relayer Command = Command{
		Name:             "relayer",
		DefaultImageURL:  HERMES_DOCKER_IMAGE_URL,
		StartCommandArgs: []string{"start"},
		StartPortArgs: []string{
			"-p", "7010:7010",
			"-p", "7011:7011",
		},
		InitCommand: "relayer init",
	}
)
