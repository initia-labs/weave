# Initia DA release smoke gate

This gate checks the default Weave build through a real rollup launch, unchanged
launch-artifact import, executor service, Initia batch submission, and submission
of a new L2 range after executor stop/start. Local tests with fake binaries and
services do not establish this result.

**Execution status (2026-10-15): NOT RUN.** No disposable service environment,
compatible devnet/staging target, or dedicated test funds were supplied. The
available host is Darwin/arm64 without a Docker daemon or Linux VM. No chain,
service, key recovery, or transaction command below was executed on that host.
The embedded scripts have only been syntax-checked. Ubuntu/macOS CI configuration
is not evidence that either runner executed this gate.

## Environment and version record

Use a disposable Ubuntu VM with a dedicated OS account, working user systemd,
D-Bus and linger. A separate macOS login session with launchd is an alternative,
but the commands below target Ubuntu (Bash, GNU `timeout` and `sha256sum`).
Changing HOME alone does not isolate the service manager. Service names are fixed:
`minitiad.service` and `opinitd.executor.service`, or `com.minitiad.daemon` and
`com.opinitd.executor.daemon` on macOS. Native rollup/executor services do not
require Docker; Docker is needed only if the chosen devnet infrastructure uses it.

Provide a designated non-mainnet L1 RPC/LCD, chain ID, compatible OPinit version,
gas prices, block timing, and dedicated faucet/test funds. The RPC must index
transactions and retain the blocks/commits used during this run. Keep L1 ports
separate from L2's 26657/1317/9090. Use a fresh rollup: initial sequence 1 uses the
artifact's L1 monitor height; an existing rollup may additionally require GraphQL
history. This procedure covers **Move** only. Other VMs and runtime versions need
their own recorded run.

Weave queries remote registry/spec and latest releases. Do not assume a new local
chain ID is supported: it must appear in the actual downloaded spec. Pin the
approved Weave commit before building. Record and retain the exact downloaded
runtime binaries, release responses, registry/spec bytes and hashes. If metadata
changes during the run, or selected binary versions differ from the approved
combination, stop and review that combination. This runbook does not change
Weave's download policy or claim to pin future remote responses.

Complete the following record with literal command output and evidence paths;
leave unexecuted cells as `NOT RUN`, never as expected values:

| Item | Observed value / evidence |
| --- | --- |
| Weave SHA, Go version, build settings and SHA256 | NOT RUN |
| OS image, architecture, dedicated account/service scope | NOT RUN |
| Initia L1 version, genesis hash, chain ID, RPC/LCD | NOT RUN |
| Move minitiad release, version, SHA256, service executable path | NOT RUN |
| opinitd release, version, SHA256, service executable path | NOT RUN |
| Registry/spec retrieval time, bytes/hash, selected spec version | NOT RUN |
| L2 chain ID, bridge ID, submitter/keyring address | NOT RUN |
| Actual bridge interval, generated max_submission_time, overrides/deadline | NOT RUN |
| Before/after marker hash/height, matching L1 tx hashes/heights and batch range | NOT RUN |
| Service logs/status, stop/start result, persistent config/key/DB checks | NOT RUN |
| Local default/tagged Initia and tagged Celestia tests: SHA, command, log/result | NOT RUN |
| Gate verdict and cleanup/staging resource owner | NOT RUN |

## Build and launch with the default Weave

Run from the approved Weave checkout in the dedicated VM. Replace the environment
placeholders below. Keep the account's real HOME; all new chain homes are under
SMOKE_ROOT. `INITIAD` must match the selected L1 and register MsgRecordBatch in its
tx decoder. Install lz4, curl, jq, Python 3 and the repository Go toolchain first.

```bash
set -euo pipefail
umask 077
SMOKE_ROOT="$(mktemp -d /tmp/weave-initia-da.XXXXXX)"
EVIDENCE="$SMOKE_ROOT/evidence"
L2_HOME="$SMOKE_ROOT/rollup"
BOT_HOME="$SMOKE_ROOT/opinit"
MARKER_HOME="$SMOKE_ROOT/marker-keyring"
mkdir -p "$EVIDENCE"
WEAVE="$SMOKE_ROOT/bin/weave"
INITIAD=/absolute/path/to/matching/initiad
L1_RPC=https://STAGING-L1-RPC
L1_LCD=https://STAGING-L1-LCD
L1_ID=SPEC-REGISTERED-L1-ID
L2_RPC=http://127.0.0.1:26657
L2_ID=unique-smoke-rollup-id
L2_DENOM=umin
EXECUTOR_URL=http://127.0.0.1:3000
export SMOKE_ROOT EVIDENCE L2_HOME BOT_HOME MARKER_HOME WEAVE INITIAD
export L1_RPC L1_LCD L1_ID L2_RPC L2_ID L2_DENOM EXECUTOR_URL

# Set this to the approved full commit before running.
APPROVED_WEAVE_SHA=REPLACE_WITH_APPROVED_FULL_SHA
test "$(git rev-parse HEAD)" = "$APPROVED_WEAVE_SHA"
test -z "$(git status --porcelain)"
GOFLAGS='' make build BUILDDIR="$SMOKE_ROOT/bin"
git rev-parse HEAD > "$EVIDENCE/weave-sha.txt"
uname -a > "$EVIDENCE/os.txt"
cat /etc/os-release >> "$EVIDENCE/os.txt"
date -u +%FT%TZ > "$EVIDENCE/start-time.txt"
go version > "$EVIDENCE/go-version.txt"
go version -m "$WEAVE" > "$EVIDENCE/weave-build-info.txt"
sha256sum "$WEAVE" "$INITIAD" > "$EVIDENCE/initial-checksums.txt"
"$INITIAD" version > "$EVIDENCE/initiad-version.txt"
curl --fail --max-time 30 "$L1_RPC/status" > "$EVIDENCE/l1-status-before.json"
curl --fail --max-time 30 "$L1_LCD/opinit/ophost/v1/params" > "$EVIDENCE/ophost-params.json"
curl --fail --max-time 30 https://raw.githubusercontent.com/initia-labs/opinit-bots/refs/heads/main/spec_version.json > "$EVIDENCE/spec_version.json"
jq -e --arg id "$L1_ID" 'has($id)' "$EVIDENCE/spec_version.json"
curl --fail --max-time 30 https://raw.githubusercontent.com/initia-labs/initia-registry/refs/heads/main/testnets/initia/chain.json > "$EVIDENCE/registry.json"
curl --fail --max-time 30 'https://api.github.com/repos/initia-labs/minimove/releases?per_page=100' > "$EVIDENCE/minimove-releases.json"
curl --fail --max-time 30 'https://api.github.com/repos/initia-labs/opinit-bots/releases?per_page=100' > "$EVIDENCE/opinit-releases.json"
sha256sum "$EVIDENCE/registry.json" "$EVIDENCE/spec_version.json" > "$EVIDENCE/metadata.sha256"
```
Inspect `weave-build-info.txt`: there must be no Celestia build tag. Record the VM
image identifier, L1 genesis hash and active spec value in the manifest. The
registry URL shown is Weave's testnet source; ensure the designated staging L1 is
compatible with the UI's registry selection. An unsupported local chain is an
environment failure, not a reason to alter the product policy.

```bash
"$WEAVE" rollup launch --minitia-dir "$L2_HOME" --opinit-dir "$BOT_HOME"
```
In the UI select the approved testnet, Move, a unique L2 chain ID, its denom, and
**Initia L1** DA. Generate dedicated system keys. Add a separate funded L2 genesis
account for the two marker transfers below and recover only that test mnemonic in
MARKER_HOME. A person must confirm the intended test keys/funding in the UI.
Keep artifact mnemonics, private configs and key screens out of public logs.

The Initia L1 funding preset allocates 2,000,000 + 2,000,000 + 1,000,000 +
2,000,000 = 7,000,000 uinit to Bridge Executor, Output Submitter, Batch Submitter
and Challenger. This is not a sufficient total-budget guarantee: account for
`.params.registration_fee`, bridge/IBC registration and bot transaction fees.
Bridge Executor creates the bridge. The marker genesis balance must cover two
transfers and their L2 fees.

For a short isolated run, choose bridge submission interval **30s** in the UI and
record this test override. Keep and separately check the generated executor
`max_submission_time=3600`. Otherwise use the actual default interval and the
longer deadline described below. A 30s run does not validate default-interval
scheduling. To additionally test the config-file launch boundary, the command is
`weave rollup launch --with-config PRIVATE_LAUNCH_JSON --vm move --minitia-dir
L2_HOME --opinit-dir BOT_HOME`; it does not replace this interactive builder gate.

```bash
ART_CONFIG="$L2_HOME/artifacts/config.json"
ARTIFACTS="$L2_HOME/artifacts/artifacts.json"
export ART_CONFIG ARTIFACTS
BRIDGE_ID="$(jq -er '.BRIDGE_ID' "$ARTIFACTS")"
BATCH_ADDR="$(jq -er '.system_keys.batch_submitter.da_address' "$ART_CONFIG")"
export BRIDGE_ID BATCH_ADDR
jq -e '.op_bridge.batch_submission_target == "INITIA"' "$ART_CONFIG"
jq '{BRIDGE_ID,EXECUTOR_L1_MONITOR_HEIGHT,EXECUTOR_L2_MONITOR_HEIGHT}' "$ARTIFACTS" > "$EVIDENCE/artifact-heights.json"
# Keep mnemonic-bearing artifact/config.json private; retain hashes and a redacted summary.
sha256sum "$ART_CONFIG" "$ARTIFACTS" > "$EVIDENCE/artifact-checksums.txt"
jq '{l1_config,l2_config,op_bridge,batch_submitter_address:.system_keys.batch_submitter.da_address}' "$ART_CONFIG" > "$EVIDENCE/launch-summary.json"
```
Read the generated service definition and record its exact executable/home. On
Ubuntu use `systemctl --user cat minitiad` and, after init, `systemctl --user cat
opinitd.executor`; on macOS inspect the corresponding plists in
`~/Library/LaunchAgents`. Do not select a stale PATH binary. Weave stores minitiad
under `~/.weave/data/mini<vm>@<version>/.../minitiad` and opinitd at
`~/.weave/data/opinitd` (which may be a symlink).

```bash
L2_BIN=/actual/service/path/to/minitiad
OPINITD="$HOME/.weave/data/opinitd"
export L2_BIN OPINITD
"$L2_BIN" version > "$EVIDENCE/minitiad-version.txt"
sha256sum "$L2_BIN" > "$EVIDENCE/minitiad-checksum.txt"
"$L2_BIN" tx bank send --help > "$EVIDENCE/l2-bank-send-help.txt"
```
## Import the unchanged artifact and start the executor

Do not pass the rollup artifact to `--with-config`: that flag expects an executor
config, not a launch artifact.

```bash
"$WEAVE" opinit init executor --minitia-dir "$L2_HOME" --opinit-dir "$BOT_HOME"
```
Choose system-key import and **Yes, prefill**. Confirm the artifact's designated
L1 RPC; set L2 RPC, denom and executor listen address to the values above. Allow
any required Oracle key only in this disposable environment. Do not replace the
artifact with a separately prepared executor JSON.

```bash
EXEC_CONFIG="$BOT_HOME/executor.json"
export EXEC_CONFIG
"$OPINITD" version --long --output json > "$EVIDENCE/opinitd-version.json"
sha256sum "$OPINITD" > "$EVIDENCE/opinitd-checksum.txt"
"$OPINITD" keys show --help > "$EVIDENCE/opinit-key-help.txt"
jq -e '.max_submission_time == 3600 and .disable_batch_submitter == false' "$EXEC_CONFIG"
jq -e '.da_node.bech32_prefix == "init" and
       .da_node.chain_id == .l1_node.chain_id and
       .da_node.rpc_address == .l1_node.rpc_address and
       .da_node.gas_price == .l1_node.gas_price' "$EXEC_CONFIG"
cp "$EXEC_CONFIG" "$EVIDENCE/executor-created.json"
"$OPINITD" keys show "$L1_ID" weave_batch_submitter --bech32 init --home "$BOT_HOME" > "$EVIDENCE/l1-batch-key-address.txt"
test "$(awk '{print $2}' "$EVIDENCE/l1-batch-key-address.txt")" = "$BATCH_ADDR"
curl --fail --max-time 30 "$L1_LCD/opinit/ophost/v1/bridges/$BRIDGE_ID" > "$EVIDENCE/bridge-before.json"
jq -e --arg a "$BATCH_ADDR" '.bridge_config.batch_info.chain_type == "INITIA" and .bridge_config.batch_info.submitter == $a' "$EVIDENCE/bridge-before.json"
# Import must not rewrite either launch artifact.
sha256sum -c "$EVIDENCE/artifact-checksums.txt"
systemctl --user cat minitiad > "$EVIDENCE/rollup-service.txt"
systemctl --user cat opinitd.executor > "$EVIDENCE/executor-service.txt"
wait_executor_ready() {
  STATUS_PHASE="$1" timeout 120 bash -c '
    until curl --fail --max-time 10 "$EXECUTOR_URL/status" \
        > "$EVIDENCE/executor-status-$STATUS_PHASE.json" && \
        jq -e ".batch_submitter != null" \
        "$EVIDENCE/executor-status-$STATUS_PHASE.json" >/dev/null; do
      sleep 2
    done
  '
}
"$WEAVE" opinit start executor --detach
wait_executor_ready before
```
The bounded readiness wait allows normal startup delay and fails after 120s.
Check the saved service arguments against L2_HOME/BOT_HOME and the hashed binaries.
Inspect the actual `.bridge_config.submission_interval` in `bridge-before.json` and
record it. Preserve `executor-created.json` before any isolated interval override;
record the precise diff if max_submission_time is also changed. No override is
normally needed when the bridge interval is 30s.

## Submit a marker and prove its inclusion in Initia DA

Confirm `keys add --recover --keyring-backend test --home ...` and `tx bank send`
using the selected minitiad's help. Recover the dedicated genesis marker account
into MARKER_HOME, and set a valid new recipient and chain-appropriate gas prices.

```bash
MARKER_KEY=smoke-marker
RECIPIENT=init1DEDICATED-RECIPIENT
L2_GAS_PRICES=0.025umin
export MARKER_KEY RECIPIENT L2_GAS_PRICES
L1_FROM="$(curl --fail --max-time 30 "$L1_RPC/status" | jq -er '.result.sync_info.latest_block_height')"
export L1_FROM
"$L2_BIN" tx bank send "$MARKER_KEY" "$RECIPIENT" "1$L2_DENOM" \
  --home "$MARKER_HOME" --keyring-backend test --node "$L2_RPC" \
  --chain-id "$L2_ID" --gas auto --gas-adjustment 1.5 --gas-prices "$L2_GAS_PRICES" \
  --broadcast-mode sync --memo weave-da-smoke-before --yes --output json > "$EVIDENCE/marker-before-broadcast.json"
MARKER_HASH="$(jq -er '.txhash' "$EVIDENCE/marker-before-broadcast.json")"
export MARKER_HASH
```
A sync broadcast is only CheckTx acceptance. The next bounded wait requires a
successful included transaction, saves its raw bytes/height and the corresponding
L2 block, and fails after 120s if inclusion cannot be established.

```bash
save_marker() {
  MARKER_PHASE="$1" MARKER_HASH="$2" timeout 120 bash -c '
    until curl --fail --max-time 10 --get "$L2_RPC/tx" \
        --data-urlencode "hash=0x$MARKER_HASH" --data-urlencode "prove=false" \
        > "$EVIDENCE/marker-$MARKER_PHASE.json" && \
        jq -e ".result.tx_result.code == 0 and (.result.height|tonumber)>0" \
        "$EVIDENCE/marker-$MARKER_PHASE.json" >/dev/null; do
      sleep 5
    done
    h=$(jq -er ".result.height" "$EVIDENCE/marker-$MARKER_PHASE.json")
    curl --fail --max-time 10 --get "$L2_RPC/block" --data-urlencode "height=$h" \
      > "$EVIDENCE/marker-$MARKER_PHASE-block.json"
  '
}
save_marker before "$MARKER_HASH"
```
The exact Comet search query is:

```bash
QUERY="message.action='/opinit.ophost.v1.MsgRecordBatch' AND tx.height>=$L1_FROM"
curl --fail --max-time 30 --get "$L1_RPC/tx_search" \
  --data-urlencode "query=\"$QUERY\"" --data-urlencode 'prove=false' \
  --data-urlencode 'page=1' --data-urlencode 'per_page=100' --data-urlencode 'order_by="asc"' > "$EVIDENCE/batches-page1.json"
```
Save the following as `$SMOKE_ROOT/collect-batches.py`. It collects **all** result
pages (at most 100 pages / 10,000 transactions), decodes successful L1 transaction
bodies, and filters the actual message bridge ID and submitter. The event only
contains submitter; do not invent a `record_batch.bridge_id` search predicate.
Exceeding the collection bound is a failure, not a truncated success.

```python
import json, os, pathlib, subprocess, urllib.parse, urllib.request
out = pathlib.Path(os.environ['EVIDENCE'])
query = "message.action='/opinit.ophost.v1.MsgRecordBatch' AND tx.height>=" + os.environ['L1_FROM']
records = []
for page in range(1, 101):
    params = dict(query=json.dumps(query), prove='false', page=str(page), per_page='100', order_by='"asc"')
    url = os.environ['L1_RPC'].rstrip('/') + '/tx_search?' + urllib.parse.urlencode(params)
    with urllib.request.urlopen(url, timeout=30) as response:
        raw = response.read()
    (out / f'batch-page-{page}.json').write_bytes(raw)
    result = json.loads(raw)['result']
    for tx in result.get('txs') or []:
        if int(tx['tx_result']['code']) != 0:
            continue
        decoded = subprocess.check_output([os.environ['INITIAD'], 'tx', 'decode', tx['tx']], timeout=30)
        body = json.loads(decoded)['body']
        (out / ('l1-tx-' + tx['hash'] + '.json')).write_bytes(decoded)
        for index, msg in enumerate(body['messages']):
            if (msg.get('@type') == '/opinit.ophost.v1.MsgRecordBatch'
                and str(msg['bridge_id']) == os.environ['BRIDGE_ID']
                and msg['submitter'] == os.environ['BATCH_ADDR']):
                records.append(dict(hash=tx['hash'], l1_height=int(tx['height']), index=index,
                                    batch_bytes=msg['batch_bytes']))
    if page * 100 >= int(result['total_count']):
        break
else:
    raise RuntimeError('bounded collection exceeded 100 pages')
(out / 'batch-records.json').write_text(json.dumps(records, indent=2))
print('matched successful MsgRecordBatch:', len(records))
```
Save the following as `$SMOKE_ROOT/check-marker-batch.py`. Run the collector
above first to filter successful L1 messages by bridge ID and submitter. The
checker excludes genesis batches, verifies the marker's successful L2 result and
hash, and validates chunk checksums, gzip framing, and the complete consecutive
block range. Each protobuf block must have the expected chain ID and height; the
marker must equal `Data.txs[index]` at its RPC-reported index, and the final commit
height must equal the batch end. Bytes appearing elsewhere in a frame do not count.

This is a focused bank-send inclusion oracle that trusts the isolated RPC results;
it does not reconstruct all runtime transactions or verify validator signatures.
Use ordinary `python3`, without `-O` or `PYTHONOPTIMIZE`: the checker rejects
optimization because it would disable assertions. On validation failure, inspect
the payload and the recorded runtime version; do not remove the checks.

```python
import base64, gzip, hashlib, json, os, pathlib, struct
if not __debug__:
    raise RuntimeError("Do not run this oracle with Python optimization enabled")
def protobuf_fields(data):
    """Decode only protobuf wire fields needed by this external smoke oracle."""
    offset = 0

    def uvarint():
        nonlocal offset
        value = 0
        for shift in range(0, 70, 7):
            if offset >= len(data):
                raise ValueError('truncated protobuf varint')
            byte = data[offset]
            offset += 1
            if shift == 63 and byte > 1:
                raise ValueError('overflowing protobuf varint')
            value |= (byte & 127) << shift
            if byte < 128:
                return value
        raise ValueError('oversize protobuf varint')

    fields = {}
    while offset < len(data):
        tag = uvarint()
        number, wire = tag >> 3, tag & 7
        if number == 0 or number >= (1 << 29):
            raise ValueError('invalid protobuf field number')
        if wire == 0:
            value = uvarint()
        elif wire in (1, 2, 5):
            length = uvarint() if wire == 2 else (8 if wire == 1 else 4)
            if offset + length > len(data):
                raise ValueError('truncated protobuf field')
            value = data[offset:offset + length]
            offset += length
        else:
            raise ValueError('unsupported protobuf wire type')
        fields.setdefault(number, []).append((wire, value))
    return fields


def one_field(fields, number, wire):
    matches = fields.get(number, [])
    if len(matches) != 1 or matches[0][0] != wire:
        raise ValueError(f'expected exactly one field {number} with wire type {wire}')
    return matches[0][1]


def verify_block_frames(frames, start, end, marker, raw_tx, rpc_block):
    # CometBFT block.proto: Block(header=1, data=2), Header(chain_id=2,
    # height=3), Data(txs=1). The last frame is the Commit(height=1).
    if not (0 < start <= end) or len(frames) != end - start + 2:
        raise ValueError('batch does not contain exactly its declared block range plus commit')
    marker_height = int(marker['height'])
    marker_index = int(marker['index'])
    rpc_txs = rpc_block['data']['txs']
    if not (0 <= marker_index < len(rpc_txs)):
        raise ValueError('marker RPC transaction index is outside its block')
    if base64.b64decode(rpc_txs[marker_index], validate=True) != raw_tx:
        raise ValueError('marker RPC index differs from actual transaction bytes')
    found = False
    for height, frame in enumerate(frames[:-1], start):
        fields = protobuf_fields(frame)
        header = protobuf_fields(one_field(fields, 1, 2))
        if one_field(header, 2, 2).decode('utf-8') != os.environ['L2_ID']:
            raise ValueError('batch block chain ID mismatch')
        if one_field(header, 3, 0) != height:
            raise ValueError('batch block height/range mismatch')
        data = protobuf_fields(one_field(fields, 2, 2))
        tx_fields = data.get(1, [])
        if any(wire != 2 for wire, _ in tx_fields):
            raise ValueError('invalid transaction field wire type')
        txs = [value for _, value in tx_fields]
        if height == marker_height:
            if marker_index >= len(txs) or txs[marker_index] != raw_tx:
                raise ValueError('marker is not the exact transaction at its block index')
            found = True
    commit = protobuf_fields(frames[-1])
    if one_field(commit, 1, 0) != end:
        raise ValueError('final commit height differs from batch end')
    if not found:
        raise ValueError('no marker block in the decoded batch')


out = pathlib.Path(os.environ['EVIDENCE'])
phase = os.environ['MARKER_PHASE']
marker = json.loads((out / f'marker-{phase}.json').read_text())['result']
assert int(marker['tx_result']['code']) == 0 and int(marker['height']) > 0
raw_tx = base64.b64decode(marker['tx'], validate=True)
assert hashlib.sha256(raw_tx).hexdigest().upper() == marker['hash'].upper()
block = json.loads((out / f'marker-{phase}-block.json').read_text())['result']['block']
assert block['header']['chain_id'] == os.environ['L2_ID']
assert int(block['header']['height']) == int(marker['height'])
assert marker['tx'] in block['data']['txs']
headers, chunks = {}, {}
for rec in json.loads((out / 'batch-records.json').read_text()):
    data = base64.b64decode(rec['batch_bytes'], validate=True)
    if data[0] == 0:
        start, end, count = struct.unpack('>QQQ', data[1:25])
        assert len(data) == 25 + count * 32
        checksums = [data[i:i+32] for i in range(25, len(data), 32)]
        headers[(start, end)] = (checksums, rec)
    elif data[0] == 1:
        start, end, index, count = struct.unpack('>QQQQ', data[1:33])
        chunks.setdefault((start, end), {})[index] = (count, data[33:], rec)
for (start, end), (checksums, header_rec) in headers.items():
    if not (start <= int(marker['height']) <= end):
        continue
    parts = chunks.get((start, end), {})
    if set(parts) != set(range(len(checksums))):
        continue  # Incomplete inclusion: retry on the next poll.
    ordered = [parts[i] for i in range(len(checksums))]
    assert all(count == len(checksums) for count, _, _ in ordered)
    assert all(hashlib.sha256(chunk).digest() == checksums[i]
               for i, (_, chunk, _) in enumerate(ordered))
    payload = gzip.decompress(b''.join(chunk for _, chunk, _ in ordered))
    frames = []
    offset = 0
    while offset < len(payload):
        assert offset + 8 <= len(payload)
        length = struct.unpack_from('<Q', payload, offset)[0]
        offset += 8
        assert offset + length <= len(payload)
        frames.append(payload[offset:offset+length])
        offset += length
    assert len(frames) >= 2  # At least one block and the final commit.
    verify_block_frames(frames, start, end, marker, raw_tx, block)
    proof = dict(phase=phase, bridge_id=os.environ['BRIDGE_ID'], submitter=os.environ['BATCH_ADDR'],
                 marker_hash=marker['hash'], marker_l2_height=int(marker['height']),
                 batch_start=start, batch_end=end, header=header_rec,
                 chunks=[rec for _, _, rec in ordered])
    (out / f'batch-proof-{phase}.json').write_text(json.dumps(proof, indent=2))
    print(json.dumps(proof, indent=2))
    break
else:
    raise SystemExit('NO COMPLETE NON-GENESIS BATCH CONTAINING THIS MARKER')
```
Use a deadline appropriate to the actual trigger. In the inspected OPinit source,
the executor must be caught up to latest; submission occurs after two-thirds of
the bridge interval **or** max_submission_time, or on size limits. With the 30s
test interval, normal few-second L1/L2 blocks and a synced executor, use 300s.
For the generated 3600s maximum with default scheduling, allow **at least 4200s**
(3600s plus inclusion/synchronization allowance), and increase that allowance for
slow staging/multiple chunks. Record the chosen deadline and reason before the
run. A lagging executor can require a longer synchronization allowance; inspect
`/status` and logs rather than declaring a short fixed timeout a product failure.

```bash
BATCH_DEADLINE=300 # Only for the documented 30s isolated bridge-interval override.
# BATCH_DEADLINE=4200 # Minimum for the unmodified 3600s maximum; adjust for staging.
export BATCH_DEADLINE
MARKER_PHASE=before timeout "$BATCH_DEADLINE" bash -c '
  until python3 "$SMOKE_ROOT/collect-batches.py" && \
        python3 "$SMOKE_ROOT/check-marker-batch.py"; do
    sleep 10
  done
'
```
The collector/checker scripts were source-reviewed and syntax-checked only; they
have not processed real chain data here. A different runtime format, missing L1
decoder registration, checksum mismatch or missing marker fails the gate and
requires review against that runtime's source. Daemon survival, a matching prefix,
or a genesis-only MsgRecordBatch is insufficient.

## Stop/start and a second L2 range

```bash
"$WEAVE" opinit stop executor
sha256sum "$EXEC_CONFIG" > "$EVIDENCE/executor-config-stopped.sha256"
"$OPINITD" keys show "$L1_ID" weave_batch_submitter --bech32 init --home "$BOT_HOME" > "$EVIDENCE/l1-batch-key-stopped.txt"
test -d "$BOT_HOME/executor.db"
```
Keep the rollup running. Do not delete or reset the executor DB. Send the second
marker while the executor is stopped; its included L2 height must be **greater
than the first proof's batch end** before restarting. If it is not, wait for L2 to
advance and submit a new distinct marker, then save that included result.

```bash
"$L2_BIN" tx bank send "$MARKER_KEY" "$RECIPIENT" "1$L2_DENOM" \
  --home "$MARKER_HOME" --keyring-backend test --node "$L2_RPC" \
  --chain-id "$L2_ID" --gas auto --gas-adjustment 1.5 --gas-prices "$L2_GAS_PRICES" \
  --broadcast-mode sync --memo weave-da-smoke-after --yes --output json > "$EVIDENCE/marker-after-broadcast.json"
MARKER_HASH="$(jq -er '.txhash' "$EVIDENCE/marker-after-broadcast.json")"
save_marker after "$MARKER_HASH"
test "$(jq -r '.result.height' "$EVIDENCE/marker-after.json")" -gt \
     "$(jq -r '.batch_end' "$EVIDENCE/batch-proof-before.json")"
"$WEAVE" opinit start executor --detach
sha256sum -c "$EVIDENCE/executor-config-stopped.sha256"
"$OPINITD" keys show "$L1_ID" weave_batch_submitter --bech32 init --home "$BOT_HOME" > "$EVIDENCE/l1-batch-key-restarted.txt"
cmp "$EVIDENCE/l1-batch-key-stopped.txt" "$EVIDENCE/l1-batch-key-restarted.txt"
wait_executor_ready restarted
MARKER_PHASE=after timeout "$BATCH_DEADLINE" bash -c '
  until python3 "$SMOKE_ROOT/collect-batches.py" && \
        python3 "$SMOKE_ROOT/check-marker-batch.py"; do
    sleep 10
  done
'
python3 - <<'CHECK'
import json, os, pathlib
p = pathlib.Path(os.environ['EVIDENCE'])
a = json.loads((p/'batch-proof-before.json').read_text())
b = json.loads((p/'batch-proof-after.json').read_text())
assert b['marker_l2_height'] > a['batch_end'] and b['batch_end'] > a['batch_end']
assert b['header']['hash'] != a['header']['hash']
print('Resumed batch covers a new marker after the earlier batch end')
CHECK
```
Require a new L1 MsgRecordBatch transaction and batch end beyond the first range.
The config hash and keyring address must stay unchanged; confirm the same
executor.db path is reused in service arguments/logs. Do not require DB file
hashes to remain identical while the executor runs. Save `/status` progress and
service logs showing batch finalization and successful inclusion:

```bash
# log streams continuously; timeout exit 124 by itself is expected here.
set +e
timeout 15s "$WEAVE" opinit log executor --n 200 > "$EVIDENCE/executor-service.log" 2>&1
LOG_EXIT=$?
set -e
printf '%s\n' "$LOG_EXIT" > "$EVIDENCE/log-command-exit.txt"
test "$LOG_EXIT" -eq 0 || test "$LOG_EXIT" -eq 124
```
On macOS substitute `shasum -a 256`/`shasum -a 256 -c`, an installed GNU timeout
(or equivalent bounded process deadline), and launchd service inspection. Do not
report a macOS gate from Ubuntu results or from compile-only checks.

## Cleanup and verdict

```bash
"$WEAVE" opinit stop executor
"$WEAVE" rollup stop
```
Preserve the evidence, selected binaries and a redacted manifest; then discard the
dedicated VM. If unloading/disabling services first, target only this run's
`minitiad` and `opinitd.executor` services. Do not prune global Docker resources,
kill unrelated processes, or remove another user's home. A staging bridge/account
and leftover test funds outlive the VM: record their owner and cleanup policy.

PASS requires both distinct successful L2 marker transactions to be covered by
complete non-genesis Initia batches for the exact bridge and submitter, including
a new range after executor stop/start, with persistent config/key/DB checks and
literal tx hashes/heights/checksums. Record failures or unexecuted steps honestly.
Link logs for the named local `TestDAFlowInitiaLaunchCompletes`,
`TestDAFlowLaunchArtifactConsumedByOPInit`, `TestDAFlowInitiaExecutorCompletes`,
`TestDAFlowInitiaDirectInitializerCompletes` (both builds) and
`TestDAFlowTaggedCelestiaExecutorCompletes` (tagged build). Their execution command
is in [README](../../README.md#contributing); none substitutes for this real gate.
If this gate is unexecuted or fails, report **local validation complete; real
Initia DA release validation incomplete** and withhold release-readiness approval.

## Source basis and limits

Commands/paths come from `cmd/rollup.go`, `cmd/opinit_bots.go`,
`models/minitia/launch.go`, `models/opinit_bots/init.go`, `registry/constants.go`
and `cosmosutils/binary.go`. The wire format, submission timing and oracle were
checked against OPinit-bots commit
[`34f9e237794107b9393d9203abf092fb6b797bab`](https://github.com/initia-labs/opinit-bots/tree/34f9e237794107b9393d9203abf092fb6b797bab):
[DA selection](https://github.com/initia-labs/opinit-bots/blob/34f9e237794107b9393d9203abf092fb6b797bab/executor/executor.go),
[batch wire format](https://github.com/initia-labs/opinit-bots/blob/34f9e237794107b9393d9203abf092fb6b797bab/executor/types/batch.go),
[submission trigger](https://github.com/initia-labs/opinit-bots/blob/34f9e237794107b9393d9203abf092fb6b797bab/executor/batchsubmitter/batch.go),
[Initia DA query](https://github.com/initia-labs/opinit-bots/blob/34f9e237794107b9393d9203abf092fb6b797bab/e2e/da_chain.go),
and [batch reconstruction test](https://github.com/initia-labs/opinit-bots/blob/34f9e237794107b9393d9203abf092fb6b797bab/e2e/batch_reconstruction_test.go).
Local OPinit v1.3.0-alpha.1 and Cosmos SDK v0.50.15-initia.0 sources were also
used for query and bank/tx-decode contracts. Upstream's example Initia v1.3.1 and
minimove v1.1.10 are source-reference versions, **not** a tested Weave runtime
combination. Verify these contracts again for the actual binaries recorded above.
