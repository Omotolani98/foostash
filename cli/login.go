package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/Omotolani98/foostash/cli/client"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newLoginCmd() *cobra.Command {
	var server string
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Log in to a Foostash server",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := LoadGlobalConfig()
			if err != nil {
				return err
			}
			if server != "" {
				cfg.Server = server
			}

			email := promptLine("Email: ")
			password := promptPassword("Password: ")

			c := client.New(cfg.Server, "")
			res, err := c.Login(email, password)
			if err != nil {
				return err
			}
			cfg.Token = res.Token
			if err := SaveGlobalConfig(cfg); err != nil {
				return err
			}
			fmt.Printf("logged in as %s (org %s)\n", res.Email, res.OrgID)
			return nil
		},
	}
	cmd.Flags().StringVar(&server, "server", "", "Server URL (e.g. https://api.foostash.dev)")
	return cmd
}

func newRegisterCmd() *cobra.Command {
	var server string
	cmd := &cobra.Command{
		Use:   "register",
		Short: "Create a new organization and user",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := LoadGlobalConfig()
			if err != nil {
				return err
			}
			if server != "" {
				cfg.Server = server
			}

			orgName := promptLine("Organization name: ")
			email := promptLine("Email: ")
			password := promptPassword("Password (min 8): ")

			c := client.New(cfg.Server, "")
			res, err := c.Register(orgName, email, password)
			if err != nil {
				return err
			}
			cfg.Token = res.Token
			if err := SaveGlobalConfig(cfg); err != nil {
				return err
			}
			fmt.Printf("registered %s (org %s)\n", res.Email, res.OrgID)
			return nil
		},
	}
	cmd.Flags().StringVar(&server, "server", "", "Server URL")
	return cmd
}

func promptLine(label string) string {
	fmt.Print(label)
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

func promptPassword(label string) string {
	fmt.Print(label)
	b, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		return ""
	}
	return string(b)
}
