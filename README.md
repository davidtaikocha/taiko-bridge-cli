# Taiko Bridge CLI

Agent-friendly CLI for Taiko Native Bridge ETH/ERC20/ERC721/ERC1155 flows.

## Build

```bash
go build ./cmd/bridge-cli
```

## Required Endpoint Flags

- `--src-rpc`
- `--dst-rpc`
- `--src-bridge`
- `--dst-bridge`
- `--src-signal-service`
- `--dst-signal-service`
- `--src-erc20-vault`
- `--src-erc721-vault`
- `--src-erc1155-vault`
- `--private-key` or `--private-key-env`

Choose source/destination endpoints to represent either direction (`L1->L2` or `L2->L1`).

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
  --src-bridge 0x... \
  --dst-bridge 0x... \
  --src-signal-service 0x... \
  --dst-signal-service 0x... \
  --src-erc20-vault 0x... \
  --src-erc721-vault 0x... \
  --src-erc1155-vault 0x... \
  --to 0xabc... \
  --value 10000000000000000 \
  --fee 1000000000000000
```

All outputs are JSON by default.
