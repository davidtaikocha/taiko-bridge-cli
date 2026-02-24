# AGENTS.md

Guidance for coding agents working in this repository.

## Project Scope

- Project: `taiko-bridge-cli`
- Binary: `bridge-cli`
- Purpose: agent-friendly Taiko bridge CLI for ETH, ERC20, ERC721, and ERC1155.
- Direction support: both `L1 -> L2` and `L2 -> L1` via explicit source/destination flags.

## Command Surface (Current)

- Pipeline commands: `claim-eth`, `claim-erc20`, `claim-erc721`, `claim-erc1155`
- Low-level commands: `send-eth`, `send-erc20`, `send-erc721`, `send-erc1155`, `wait-ready`, `claim`, `status`
- Agent helpers: `agent exit-codes`, `schema`
- Compatibility aliases: `bridge-eth`, `bridge-erc20`, `bridge-erc721`, `bridge-erc1155` (alias to matching `claim-*` command)

## Configuration Rules

- Use flags only. Do not reintroduce YAML config files.
- Required global flags are source/destination RPC + contract addresses.
- Private key is loaded via `--private-key` or `--private-key-env`.
- Keep machine-friendly JSON output as the default (`--format json`).

## Readiness and Claiming Rules

- Assume destination L2 is already Shasta-ready.
- `wait-ready` must:
1. find source `MessageSent` block,
2. scan destination `CheckpointSaved` history,
3. pick latest qualifying checkpoint (`checkpoint.blockNumber >= source block`),
4. build proof and run `isMessageReceived`,
5. return ready immediately when true, else poll until timeout.
- `claim-*` should keep optimistic auto-retry behavior (attempt claim immediately, then poll/retry until success or timeout).

## Code Layout

- Entrypoint: `cmd/bridge-cli/main.go`
- Commands: `internal/cmd/*`
- Send logic: `internal/bridge/send.go`
- Readiness logic: `internal/ready/wait.go`
- Proof logic: `internal/proof/proof.go`
- Claim logic: `internal/claim/process.go`
- Contract bindings: `internal/bindings/*`
  - Signal service binding path is `internal/bindings/signalservice`.

## Generated Code

- `internal/bindings/*` are generated bindings.
- Do not hand-edit generated files unless explicitly requested.

## Documentation and Commenting

- Keep function/method/struct comments in place.
- New/modified Go code should include clear doc comments for:
  - exported and non-trivial functions/methods,
  - structs,
  - struct fields when field purpose is not obvious.

## Validation Checklist (Before claiming completion)

1. `gofmt -w` on changed Go files.
2. `go test ./...`
3. If CLI surface changed: `go run ./cmd/bridge-cli --help`
4. If command behavior changed: update `README.md` examples/flags accordingly.

## Safety Constraints

- Preserve existing command names and JSON fields unless a change is explicitly requested.
- Keep `bridge-*` compatibility aliases unless explicitly removed by user request.
- Avoid destructive git commands.
