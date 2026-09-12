package cli

import (
	"github.com/amansk/parkmobile-pp-cli/internal/exitcode"
	"github.com/spf13/cobra"
)

func newAccountCmd(opt *Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "account",
		Short: "Account profile and payment methods (no secrets)",
	}
	cmd.AddCommand(newAccountMeCmd(opt))
	cmd.AddCommand(newVehiclesCmd(opt))
	cmd.AddCommand(newPaymentMethodsCmd(opt))
	return cmd
}

func newAccountMeCmd(opt *Options) *cobra.Command {
	return &cobra.Command{
		Use:   "me",
		Short: "Show account profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := opt.newClient()
			if err != nil {
				return err
			}
			if c.Session == nil || c.Session.CookieHeader() == "" {
				return exitcode.Authf("authenticated session required; run auth login")
			}
			profile, err := c.GetAccount()
			if err != nil {
				return err
			}
			return writeOut(cmd, opt, profile)
		},
	}
}

func newVehiclesCmd(opt *Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vehicles",
		Short: "List registered vehicles",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List vehicles (ids + plates only)",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := opt.newClient()
			if err != nil {
				return err
			}
			if c.Session == nil || c.Session.CookieHeader() == "" {
				return exitcode.Authf("authenticated session required; run auth login")
			}
			vehicles, err := c.ListVehicles()
			if err != nil {
				return err
			}
			return writeOut(cmd, opt, map[string]any{"vehicles": vehicles, "count": len(vehicles)})
		},
	})
	return cmd
}

func newPaymentMethodsCmd(opt *Options) *cobra.Command {
	return &cobra.Command{
		Use:   "payment-methods",
		Short: "List saved payment methods (ids + last4 only)",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := opt.newClient()
			if err != nil {
				return err
			}
			if c.Session == nil || c.Session.CookieHeader() == "" {
				return exitcode.Authf("authenticated session required; run auth login")
			}
			methods, err := c.ListPaymentMethods()
			if err != nil {
				return err
			}
			return writeOut(cmd, opt, map[string]any{"payment_methods": methods, "count": len(methods)})
		},
	}
}
