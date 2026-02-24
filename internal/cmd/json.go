package cmd

import (
	"os"

	"github.com/davidcai/taiko-bridge-cli/internal/outfmt"
)

func printJSON(format string, v any) error {
	return outfmt.Printer{Format: format, Out: os.Stdout}.Emit(v)
}
