package outfmt

import (
	"encoding/json"
	"fmt"
	"io"
)

type Printer struct {
	Format string
	Out    io.Writer
}

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
