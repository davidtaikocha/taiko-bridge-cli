# Taiko Bridge CLI

Agent-friendly CLI for Taiko Native Bridge ETH/ERC20/ERC721/ERC1155 flows.

## Build

```bash
go build ./cmd/bridge-cli
```

## Required Flags

- `--src-rpc`
- `--dst-rpc`
- `--private-key` or `--private-key-env`

Contract addresses are auto-resolved from source/destination chain IDs for Taiko Mainnet and Taiko Hoodi.

Optional override flags (only needed for custom deployments):
- `--src-bridge`, `--dst-bridge`
- `--src-signal-service`, `--dst-signal-service`
- `--src-erc20-vault`, `--src-erc721-vault`, `--src-erc1155-vault`

Choose source/destination RPC endpoints to represent either direction (`L1->L2` or `L2->L1`).

## Command Surface

- Pipeline: `claim-eth`, `claim-erc20`, `claim-erc721`, `claim-erc1155`
- Low-level: `send-eth`, `send-erc20`, `send-erc721`, `send-erc1155`, `wait-ready`, `claim`, `status`
- Agent helpers: `agent exit-codes`, `schema`
- Compatibility aliases: `bridge-eth`, `bridge-erc20`, `bridge-erc721`, `bridge-erc1155`

## Example

```bash
PRIVATE_KEY=0x... ./bridge-cli claim-eth \
  --src-rpc https://l1-rpc \
  --dst-rpc https://l2-rpc \
  --to 0xabc... \
  --value 10000000000000000 \
  --fee 1000000000000000
```

All outputs are JSON by default.
