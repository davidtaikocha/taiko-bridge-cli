package exitcodes

const (
	// OK indicates successful execution.
	OK = 0
	// ConfigError indicates missing or invalid runtime configuration.
	ConfigError = 2
	// Validation indicates invalid user input.
	Validation = 3
	// Timeout indicates the command timed out before completion.
	Timeout = 4
	// TxReverted indicates an on-chain transaction reverted.
	TxReverted = 5
	// NotReady indicates message state is not claim-ready yet.
	NotReady = 6
	// RPCOrProof indicates RPC, proof generation, or chain-call failures.
	RPCOrProof = 7
	// Unexpected indicates an unclassified failure.
	Unexpected = 10
)
