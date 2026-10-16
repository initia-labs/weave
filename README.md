![image (1)](https://github.com/user-attachments/assets/74ca0671-a4b7-48bc-aded-cca1816a418d)

# Weave

Weave is a CLI tool designed to make working with Initia and its Interwoven Rollups easier. Instead of dealing with multiple tools and extensive documentation,
developers can use a single command-line interface for the entire development and deployment workflow.

Its primary purpose is to solve several key challenges:

1. **Infrastructure Management:** Weave can handle all critical infrastructure components within the Interwoven Rollup ecosystem:
   - Initia node setup and management (including state sync and chain upgrade management)
   - Rollup deployment and configuration
   - OPinit bots setup for the Optimistic bridge
   - IBC Relayer setup between Initia L1 and your Rollup
2. **Built for both local development and production deployments:** Weave provides
   - Interactive guided setup for step-by-step configuration and
   - Configuration file support for automated deployments
3. **Developer Experience:** Not only does it consolidate multiple complex operations into a single CLI tool, but it also changes how you interact with the tool to set up your configuration.

## Prerequisites

- Operating System: **Linux, macOS**
- Go **1.24.0** or higher when building from source; `go.mod` declares toolchain **go1.24.11**
- LZ4 compression tool
  - For macOS: `brew install lz4`
  - For Ubuntu/Debian: `apt-get install lz4`
  - For other Linux distributions: Use your package manager to install lz4

> **Important:** While Weave can run as root, it does not support switching users via commands like `sudo su ubuntu` or `su - someuser`. Instead, directly SSH or log in as the user you intend to run Weave with. For example:
>
> ```bash
> ssh ubuntu@your-server    # Good: Direct login as ubuntu user
> ssh root@your-server     # Good: Direct login as root
> ```
>
> This ensures proper handling of user-specific configurations and paths.

## Get started

👉 https://docs.initia.xyz/developers/developer-guides/tools/clis/weave-cli/installation

## Data availability

The default build supports **Initia DA** for new rollup launches, executor initialization
(including reusing an existing executor config), and Batch Submitter key creation or import.
These operations reject Celestia DA with an explicit error. Existing Celestia service
`start`, `stop`, `restart`, and `log` commands, Gas Station addresses, and balance queries
remain available. Existing Celestia configs are not migrated to Initia.

New configs must use exact DA values: rollup `batch_submission_target` is `INITIA` or
`CELESTIA`, and executor `da_node.bech32_prefix` is `init` or `celestia`. Missing, unknown,
case-altered, or whitespace-padded values are rejected. Artifact imports also validate
DA address consistency; legacy artifacts without a target use their valid DA address,
or their L1 address when no DA address is present.

Developers who need new Celestia DA operations can explicitly build from source:

```sh
make build                            # default: new Initia DA operations
GOFLAGS='-tags=celestia' make build   # opt in to new Celestia DA operations
```

The build tag is a compile-time option. Both builds retain the existing Celestia support
code and accept valid Initia DA configurations.

## Usage data collection

By default, Weave collects non-identifiable usage data to help improve the product. If you prefer not to share this data, you can opt out by running the following command:

```bash
weave analytics disable
```

## Contributing

We welcome contributions! Please feel free to submit a pull request.

Run unit tests and lint for both builds with a Go 1.24-compatible golangci-lint v1.64.8:

```sh
go test -count=1 ./...
go test -count=1 -tags=celestia ./...
make lint
GOFLAGS='-tags=celestia' make lint

# Focused DA policy, successful launch/init flows, and service dispatch
go test -count=1 ./common ./models/minitia ./models/opinit_bots ./cmd -run '^Test(DAPolicy|DAFlow|ServiceLifecycle)'
go test -count=1 -tags=celestia ./common ./models/minitia ./models/opinit_bots ./cmd -run '^Test(DAPolicy|DAFlow|ServiceLifecycle)'

# Builds and exercises both real CLI binaries in isolated temporary homes
go test -count=1 -tags=e2e ./cmd -run '^TestCelestiaPolicyE2E$'
```

The broad `go test -count=1 -tags=integration ./...` suite downloads binaries and changes
home directories and services; run it only on a disposable runner. It retains existing
skipped and disabled scenarios. Local flow tests replace external binaries and OS services;
release validation additionally requires the [actual Initia batch submission and restart smoke gate](docs/testing/initia-da-smoke.md).
