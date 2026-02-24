package outfmt

import (
	"encoding/json"
	"fmt"
	"io"
)

// Printer writes command outputs in the requested format.
type Printer struct {
	// Format controls serialization mode, typically json or text.
	Format string
	// Out is the destination writer for output lines.
	Out io.Writer
}

// Emit writes v to Out in the configured format.
func (p Printer) Emit(v any) error {
	if p.Out == nil {
		return fmt.Errorf("nil output writer")
	}
	if p.Format == "" || p.Format == "json" {
		b, err := json.Marshal(v)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(p.Out, string(b))
		return err
	}
	_, err := fmt.Fprintln(p.Out, v)
	return err
}
