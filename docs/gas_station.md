# Gas Station

The Gas Station is a dedicated account used by Weave to fund critical infrastructure components of the Interwoven stack. It distributes funds to essential services like [OPinit Bots](/nodes-and-rollups/deploying-rollups/opinit-bots/introduction) (including Bridge Executor, Output Submitter, Batch Submitter, and Challenger) and the [IBC relayer](https://tutorials.cosmos.network/academy/2-cosmos-concepts/13-relayer-intro.html) to ensure smooth operation of the network.

This is essential for seamless operation with Weave as it eliminates the need for manual fund distribution.

> While Weave requires your consent for all fund transfers, using a separate account prevents any potential misuse of an existing account. We strongly recommend creating a new dedicated account for Gas Station use rather than using an existing account

## Setting up the Gas Station

```bash
weave gas-station setup
```

You can either import an existing mnemonic or have Weave generate a new one.
Once setup is complete, you'll see two addresses in `init` and `celestia` format.

> While the Gas Station addresses for Celestia and the Initia ecosystem will be different, both are derived from the same mnemonic that you entered.

Then fund the account with at least 10 INIT tokens to support the necessary components. If you're planning to use Celestia as your Data Availability Layer, you'll also need to fund the account with `TIA` tokens.

> For testnet operations: - Get testnet `INIT` tokens from the [Initia faucet](https://faucet.testnet.initia.xyz/) - Get testnet `TIA` tokens from the [Celestia faucet](https://docs.celestia.org/how-to-guides/mocha-testnet#mocha-testnet-faucet)

## Viewing Gas Station Information

```bash
weave gas-station show
```

This command displays the addresses and current balances of the Gas Station account in both `init` and `celestia` bech32 formats.
