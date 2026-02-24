# Taiko Bridge CLI (`tbc`)

Agent-friendly CLI for Taiko Bridge ETH/ERC20/ERC721/ERC1155 flows on Shasta.

## Build

```bash
go build ./cmd/tbc
```

## Config

Create `tbc.yaml`:

```yaml
profiles:
  l1_to_l2:
    src:
      rpc_url: https://l1-rpc
      bridge: 0x...
      signal_service: 0x...
      erc20_vault: 0x...
      erc721_vault: 0x...
      erc1155_vault: 0x...
    dest:
      rpc_url: https://l2-rpc
      bridge: 0x...
      signal_service: 0x...
      erc20_vault: 0x...
      erc721_vault: 0x...
      erc1155_vault: 0x...

  l2_to_l1:
    src:
      rpc_url: https://l2-rpc
      bridge: 0x...
      signal_service: 0x...
      erc20_vault: 0x...
      erc721_vault: 0x...
      erc1155_vault: 0x...
    dest:
      rpc_url: https://l1-rpc
      bridge: 0x...
      signal_service: 0x...
      erc20_vault: 0x...
      erc721_vault: 0x...
      erc1155_vault: 0x...
```

## Command Surface

- Pipeline: `claim-eth`, `claim-erc20`, `claim-erc721`, `claim-erc1155`
- Low-level: `send-eth`, `send-erc20`, `send-erc721`, `send-erc1155`, `wait-ready`, `claim`, `status`
- Agent helpers: `agent exit-codes`, `schema`
- Compatibility aliases: `bridge-eth`, `bridge-erc20`, `bridge-erc721`, `bridge-erc1155`

## Example

```bash
PRIVATE_KEY=0x... ./tbc claim-eth \
  --config ./tbc.yaml \
  --profile l1_to_l2 \
  --to 0xabc... \
  --value 10000000000000000 \
  --fee 1000000000000000
```

All outputs are JSON by default.
