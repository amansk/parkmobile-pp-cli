package cli

import (
	"github.com/amansk/parkmobile-pp-cli/internal/exitcode"
	"github.com/spf13/cobra"
)

func newZonesCmd(opt *Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "zones",
		Short: "Zone parking lookup",
	}
	cmd.AddCommand(newZonesGetCmd(opt))
	return cmd
}

func newZonesGetCmd(opt *Options) *cobra.Command {
	var zoneCode string
	var duration int
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get zone info, rates, and stoppable/max duration flags",
		RunE: func(cmd *cobra.Command, args []string) error {
			if zoneCode == "" {
				return exitcode.Usagef("--zone is required")
			}
			c, err := opt.newClient()
			if err != nil {
				return err
			}
			if c.Session == nil || c.Session.CookieHeader() == "" {
				return exitcode.Authf("authenticated session required; run auth login")
			}
			zone, err := c.GetZone(zoneCode)
			if err != nil {
				return err
			}
			out := map[string]any{
				"zone": zone,
			}
			if duration > 0 {
				quote, qerr := c.GetPriceQuote(zoneCode, duration, "")
				if qerr == nil {
					out["price_quote"] = quote
				} else {
					out["price_quote_error"] = qerr.Error()
				}
			}
			return writeOut(cmd, opt, out)
		},
	}
	cmd.Flags().StringVar(&zoneCode, "zone", "", "Zone/signage code from meter signage")
	cmd.Flags().IntVar(&duration, "duration-minutes", 0, "Optional duration for inline price quote")
	return cmd
}
