![image (1)](https://github.com/user-attachments/assets/74ca0671-a4b7-48bc-aded-cca1816a418d)

# Weave

Weave is a CLI tool designed to make working with Initia and its Interwoven Rollups easier. Instead of dealing with multiple tools and extensive documentation,
developers can use a single command-line interface for the entire development and deployment workflow.

Its primary purpose is to solve several key challenges:

1. **Infrastructure Management:** Weave can handles all critical infrastructure components within the Interwoven Rollup ecosystem:
   - Initia node setup and management (including state sync and chain upgrade management)
   - Rollup deployment and configuration
   - OPinit bots setup for the Optimistic bridge
   - IBC Relayer setup between Initia L1 and your Rollup
2. **Built for both local development and production deployments:** Weave provides
   - Interactive guided setup for step-by-step configuration and
   - Configuration file support for automated deployments
3. **Developer Experience:** Not only it consolidates multiple complex operations into a single CLI tool, but it also changes how you interact with the tool to setup your configuration.

## Prerequisites

- Operating System: **Linux, macOS**
- Go **v1.23** or higher when building from scratch

## Installation

### On macOS

Install _Weave_ via [Homebrew](https://brew.sh/):

```bash
brew install initia-labs/tap/weave
```

### On Linux

Install _Weave_ by downloading the appropriate binary for your architecture using `wget`:

**For x86_86 (amd64)**

```bash
wget https://github.com/initia-labs/weave/releases/download/v0.1.2/weave-0.1.2-linux-amd64.tar.gz
tar -xvf weave-0.1.2-linux-amd64.tar.gz
```

**For arm64**

```bash
wget https://github.com/initia-labs/weave/releases/download/v0.1.2/weave-0.1.2-linux-arm64.tar.gz
tar -xvf weave-0.1.2-linux-arm64.tar.gz
```

### Building from Scratch

To build _Weave_ from source, you will need a working Go environment and `make`. Follow these steps:

```bash
git clone https://github.com/initia-labs/weave.git
cd weave
git checkout tags/v0.1.2
make install
```

### Download Pre-built binaries

Go to the [Releases](https://github.com/initia-labs/weave/releases) page and download the binary for your operating system.

### Verify Installation

```bash
weave version
```
This should return the version of the Weave binary you have installed. Example output:

```bash
v0.1.2
```


## Quick Start

To get started with Weave, run
```bash
weave init
```
It will ask you to setup the [Gas Station](/docs/gas_station.md) account and ask which infrastructure you want to setup.
After that, Weave will guide you through the setup process step-by-step.

## Usage

1. [Bootstrapping Initia Node](/docs/initia_node.md)
2. [Launch a new rollup](/docs/rollup_launch.md)
3. [Setting up IBC relayer](/docs/relayer.md)
4. [Setting up OPinit bots](/docs/opinit_bots.md)

## Usage data collection

By default, Weave collects non-identifiable usage data to help improve the product. If you prefer not to share this data, you can opt out by running the following command:
```bash
weave analytics disable
```

## Contributing

We welcome contributions! Please feel free to submit a pull request.
