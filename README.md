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
- Go **v1.23** or higher when building from scratch
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

## Usage data collection

By default, Weave collects non-identifiable usage data to help improve the product. If you prefer not to share this data, you can opt out by running the following command:

```bash
weave analytics disable
```

## Contributing

We welcome contributions! Please feel free to submit a pull request.
