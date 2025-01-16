# Running IBC Relayer

An IBC relayer is a software component that facilitates communication between two distinct blockchain networks that support the Inter-Blockchain Communication (IBC) protocol.

It is required for built-in oracle, Minitswap, and other cross-chain services to function with your rollup.

While setting up a relayer is traditionally one of the most complex tasks in the Cosmos ecosystem, 
Weave simplifies this process significantly, reducing the typical 10+ step IBC Relayer setup to just a few simple steps.

> Weave only supports IBC relayer setup between Initia L1 and Interwoven Rollups. Setting up relayers between other arbitrary networks is not supported.

> Currently, Weave only supports the `Hermes` relayer. For detailed information about Hermes, please refer to the [Hermes documentation](https://github.com/informalsystems/hermes).

## Setting up

```bash
weave relayer init
```
This command will guide you through 2 major parts of the relayer setup:
- Setting up networks and channels to relay messages between
- Setting up the account responsible for relaying messages


For the former, Weave will present you with three options:
1. Configure channels between Initia L1 and a whitelisted Rollup (those available in [Initia Registry](https://github.com/initia-labs/initia-registry)
2. Configure using artifacts from `weave rollup launch` (recommended for users who have just launched their rollup)
3. Configure manually

As for the latter, Weave will ask whether you want to use OPinit Challenger bot account for the relayer. This is recommended as it is exempted from gas fees on the rollup and able to stop other relayers from relaying when it detects a malicious message coming from it.

> Relayer requires funds to relay messages between Initia L1 and your rollup (if it's not in the fee whitelist). If Weave detects that your account does not have enough funds, Weave will ask you to fund via Gas Station.

> For advanced configuration options, you can refer to the [Hermes Configuration Guide](https://hermes.informal.systems/documentation/configuration/configure-hermes.html) and customize the relayer's configuration file located at `~/.hermes/config.toml`.

## Running Relayer

### Start the relayer

```bash
weave relayer start
```

Specify `--detach` or `-d` to run in the background.
Specify `--update-client false` to disable update IBC clients on relayer starts. Default to `true` 

### Stop the relayer

```bash
weave relayer stop
```

### Restart the relayer

```bash
weave relayer restart
```

### See the logs

```bash
weave relayer log
```

## Help

To see all the available commands: 
```bash
weave relayer --help
```
