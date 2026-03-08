package output

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/itchyny/gojq"
	"github.com/spf13/cobra"
)

type Output struct {
	cmd *cobra.Command
}

func New(cmd *cobra.Command) *Output {
	return &Output{cmd: cmd}
}

func (o *Output) Print(data any) error {
	jsonFlag, _ := o.cmd.Flags().GetBool("json")
	jqExpr, _ := o.cmd.Flags().GetString("jq")
	if jqExpr != "" || jsonFlag {
		encoded, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return err
		}
		if jqExpr == "" {
			fmt.Fprintln(os.Stdout, string(encoded))
			return nil
		}
		var payload any
		if err := json.Unmarshal(encoded, &payload); err != nil {
			return err
		}
		query, err := gojq.Parse(jqExpr)
		if err != nil {
			return err
		}
		iter := query.Run(payload)
		for {
			v, ok := iter.Next()
			if !ok {
				break
			}
			if err, ok := v.(error); ok {
				return err
			}
			fmt.Fprintln(os.Stdout, v)
		}
		return nil
	}

	fmt.Fprintf(os.Stdout, "%+v\n", data)
	return nil
}
