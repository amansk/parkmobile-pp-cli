package cli

import (
	"github.com/amansk/parkmobile-pp-cli/internal/exitcode"
	"github.com/spf13/cobra"
)

func newSessionsCmd(opt *Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sessions",
		Short: "List and inspect parking sessions",
	}
	cmd.AddCommand(newSessionsListCmd(opt))
	cmd.AddCommand(newSessionsGetCmd(opt))
	return cmd
}

func newSessionsListCmd(opt *Options) *cobra.Command {
	var page, pageSize int
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List active and recent parking sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := opt.newClient()
			if err != nil {
				return err
			}
			if c.Session == nil || c.Session.CookieHeader() == "" {
				return exitcode.Authf("authenticated session required; run auth login")
			}
			sessions, err := c.ListSessions(page, pageSize)
			if err != nil {
				return err
			}
			return writeOut(cmd, opt, map[string]any{"sessions": sessions, "count": len(sessions)})
		},
	}
	cmd.Flags().IntVar(&page, "page", 0, "Page number")
	cmd.Flags().IntVar(&pageSize, "page-size", 0, "Page size")
	return cmd
}

func newSessionsGetCmd(opt *Options) *cobra.Command {
	return &cobra.Command{
		Use:   "get <session-id>",
		Short: "Get one parking session by id",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := opt.newClient()
			if err != nil {
				return err
			}
			if c.Session == nil || c.Session.CookieHeader() == "" {
				return exitcode.Authf("authenticated session required; run auth login")
			}
			session, err := c.GetSession(args[0])
			if err != nil {
				return err
			}
			return writeOut(cmd, opt, session)
		},
	}
}
