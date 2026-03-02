package cmd

import (
	"io"

	"github.com/davidcai/taiko-bridge-cli/internal/outfmt"
)

// printJSON emits a value using the configured output format.
func printJSON(format string, out io.Writer, v any) error {
	return outfmt.Printer{Format: format, Out: out}.Emit(v)
}
