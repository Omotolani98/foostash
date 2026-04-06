package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newDiffCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "diff LEFT_ENV RIGHT_ENV",
		Short: "Show which secret keys differ between two environments",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			proj, err := LoadProjectConfig()
			if err != nil {
				return fmt.Errorf("no .foostash.yaml in current directory (run `foostash init` first)")
			}
			c := mustClient()
			res, err := c.Diff(proj.Project, args[0], args[1])
			if err != nil {
				return err
			}

			printList := func(label string, keys []string) {
				if len(keys) == 0 {
					return
				}
				fmt.Printf("\n%s:\n", label)
				for _, k := range keys {
					fmt.Printf("  %s\n", k)
				}
			}
			fmt.Printf("diff %s <- %s vs %s ->\n", proj.Project, args[0], args[1])
			printList(fmt.Sprintf("only in %s", args[0]), res.OnlyLeft)
			printList(fmt.Sprintf("only in %s", args[1]), res.OnlyRight)
			printList("different values", res.DifferentValues)
			printList("identical", res.Identical)
			return nil
		},
	}
}
