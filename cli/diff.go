package cli

import (
	"fmt"

	"github.com/Omotolani98/foostash/internal/diff"
	"github.com/spf13/cobra"
)

func newDiffCmd(app *App) *cobra.Command {
	var format string
	var mask, onlyChanges bool

	cmd := &cobra.Command{
		Use:   "diff LEFT_ENV RIGHT_ENV",
		Short: "Compare two environments",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			proj, err := loadProjectConfig()
			if err != nil {
				return fmt.Errorf("no .foostash.yaml in current directory (run `foostash init` first)")
			}

			leftName, rightName := args[0], args[1]

			left, err := app.Secrets.Pull(proj.Project, leftName, false)
			if err != nil {
				return fmt.Errorf("load %s: %w", leftName, err)
			}
			right, err := app.Secrets.Pull(proj.Project, rightName, false)
			if err != nil {
				return fmt.Errorf("load %s: %w", rightName, err)
			}

			result := diff.Compare(left, right)

			switch format {
			case "table", "":
				fmt.Print(diff.FormatTable(result, leftName, rightName, mask, onlyChanges))
			case "json":
				b, err := diff.FormatJSON(result, leftName, rightName)
				if err != nil {
					return err
				}
				fmt.Println(string(b))
			default:
				return fmt.Errorf("unknown format: %s", format)
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&format, "format", "f", "table", "Output format: table|json")
	cmd.Flags().BoolVar(&mask, "mask", false, "Redact secret values")
	cmd.Flags().BoolVar(&onlyChanges, "only-changes", false, "Show only differences")
	return cmd
}
