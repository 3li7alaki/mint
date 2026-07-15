package receiptcmd

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"

	"mint/internal/receipt"
)

type Flags struct {
	JSON bool
}

func Run(root string, args []string, flags Flags, stdout io.Writer) (int, error) {
	if len(args) != 2 || (args[0] != "show" && args[0] != "verify") {
		return 1, fmt.Errorf("Usage: mint receipt show|verify <path> [--json]")
	}
	path := args[1]
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}
	record, err := receipt.Read(path)
	if err != nil {
		return 1, err
	}
	if args[0] == "show" {
		return printJSON(stdout, record)
	}
	validation := receipt.Validate(root, record)
	if flags.JSON {
		if _, err := printJSON(stdout, validation); err != nil {
			return 1, err
		}
	} else if validation.Current {
		fmt.Fprintf(stdout, "current %s\n", validation.ReceiptDigest)
	} else {
		fmt.Fprintf(stdout, "stale %s - %s\n", validation.ReceiptDigest, validation.Reason)
	}
	if !validation.Valid || !validation.Current {
		return 1, nil
	}
	return 0, nil
}

func printJSON(stdout io.Writer, value any) (int, error) {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return 1, err
	}
	_, err = fmt.Fprintln(stdout, string(b))
	return 0, err
}
