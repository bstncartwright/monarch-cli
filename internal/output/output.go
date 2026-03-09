package output

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"
)

type Table struct {
	Headers []string
	Rows    [][]string
}

func WriteJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func WriteTable(w io.Writer, table Table) error {
	if len(table.Headers) == 0 {
		return fmt.Errorf("table output requires headers")
	}
	tw := tabwriter.NewWriter(w, 0, 8, 2, ' ', 0)
	for i, header := range table.Headers {
		if i > 0 {
			_, _ = fmt.Fprint(tw, "\t")
		}
		_, _ = fmt.Fprint(tw, header)
	}
	_, _ = fmt.Fprint(tw, "\n")
	for _, row := range table.Rows {
		for i, cell := range row {
			if i > 0 {
				_, _ = fmt.Fprint(tw, "\t")
			}
			_, _ = fmt.Fprint(tw, cell)
		}
		_, _ = fmt.Fprint(tw, "\n")
	}
	return tw.Flush()
}
