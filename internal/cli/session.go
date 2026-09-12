package cli

import (
	"strconv"

	"github.com/amansk/parkmobile-pp-cli/internal/client"
	"github.com/amansk/parkmobile-pp-cli/internal/exitcode"
	"github.com/spf13/cobra"
)

const (
	startConfirmPhrase  = "START PARKMOBILE SESSION"
	extendConfirmPhrase = "EXTEND PARKMOBILE SESSION"
	stopConfirmPhrase   = "STOP PARKMOBILE SESSION"
)

func newSessionCmd(opt *Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "session",
		Short: "Start, extend, or stop zone parking (mutations hard-gated)",
	}
	cmd.AddCommand(newSessionStartCmd(opt))
	cmd.AddCommand(newSessionExtendCmd(opt))
	cmd.AddCommand(newSessionStopCmd(opt))
	return cmd
}

func newSessionStartCmd(opt *Options) *cobra.Command {
	var zoneCode string
	var duration, vehicleID, billingMethodID int
	var spaceNumber, orderToken string
	var enableLive, ownerApproved bool
	var confirm string

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start zone parking (hard-gated) or use 'start preview' for quote only",
		RunE: func(cmd *cobra.Command, args []string) error {
			if zoneCode == "" || duration <= 0 {
				return exitcode.Usagef("--zone and --duration-minutes are required")
			}
			if !enableLive || !ownerApproved || confirm != startConfirmPhrase {
				return exitcode.Usagef("refusing live parking: require --enable-live-parking --owner-approved --confirm %q", startConfirmPhrase)
			}
			c, err := opt.newClient()
			if err != nil {
				return err
			}
			if c.Session == nil || c.Session.CookieHeader() == "" {
				return exitcode.Authf("authenticated session required; run auth login")
			}
			out, err := c.StartSession(client.StartSessionInput{
				ZoneCode:        zoneCode,
				DurationMinutes: duration,
				VehicleID:       vehicleID,
				BillingMethodID: billingMethodID,
				SpaceNumber:     spaceNumber,
				OrderToken:      orderToken,
			})
			if err != nil {
				return err
			}
			return writeOut(cmd, opt, map[string]any{"started": true, "result": out})
		},
	}
	addStartFlags(cmd, &zoneCode, &duration, &vehicleID, &billingMethodID, &spaceNumber)
	cmd.Flags().StringVar(&orderToken, "order-token", "", "Optional order token from preview")
	cmd.Flags().BoolVar(&enableLive, "enable-live-parking", false, "Explicit opt-in to charge a payment method")
	cmd.Flags().BoolVar(&ownerApproved, "owner-approved", false, "Explicit owner approval for this session")
	cmd.Flags().StringVar(&confirm, "confirm", "", "Must be exactly: "+startConfirmPhrase)
	cmd.AddCommand(newSessionStartPreviewCmd(opt))
	return cmd
}

func newSessionStartPreviewCmd(opt *Options) *cobra.Command {
	var zoneCode string
	var duration, vehicleID, billingMethodID int
	var spaceNumber string
	cmd := &cobra.Command{
		Use:   "preview",
		Short: "Preview a parking quote without charging",
		RunE: func(cmd *cobra.Command, args []string) error {
			if zoneCode == "" || duration <= 0 {
				return exitcode.Usagef("--zone and --duration-minutes are required")
			}
			c, err := opt.newClient()
			if err != nil {
				return err
			}
			preview, err := c.StartPreview(client.StartPreviewInput{
				ZoneCode:        zoneCode,
				DurationMinutes: duration,
				VehicleID:       vehicleID,
				BillingMethodID: billingMethodID,
				SpaceNumber:     spaceNumber,
			})
			if err != nil {
				return err
			}
			return writeOut(cmd, opt, preview)
		},
	}
	addStartFlags(cmd, &zoneCode, &duration, &vehicleID, &billingMethodID, &spaceNumber)
	return cmd
}

func newSessionExtendCmd(opt *Options) *cobra.Command {
	var sessionID int
	var duration, billingMethodID, timeBlockID int
	var enableLive, ownerApproved bool
	var confirm string
	cmd := &cobra.Command{
		Use:   "extend",
		Short: "Extend an active parking session (hard-gated)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if sessionID <= 0 || duration <= 0 {
				return exitcode.Usagef("--session-id and --duration-minutes are required")
			}
			if !enableLive || !ownerApproved || confirm != extendConfirmPhrase {
				return exitcode.Usagef("refusing live extend: require --enable-live-parking --owner-approved --confirm %q", extendConfirmPhrase)
			}
			c, err := opt.newClient()
			if err != nil {
				return err
			}
			if c.Session == nil || c.Session.CookieHeader() == "" {
				return exitcode.Authf("authenticated session required; run auth login")
			}
			sess, err := c.GetSession(strconv.Itoa(sessionID))
			if err != nil {
				return err
			}
			if !sess.CanExtend {
				return exitcode.Usagef("session %d cannot be extended (can_extend=false)", sessionID)
			}
			out, err := c.ExtendSession(client.ExtendSessionInput{
				SessionID:       sessionID,
				DurationMinutes: duration,
				BillingMethodID: billingMethodID,
				TimeBlockID:     timeBlockID,
			})
			if err != nil {
				return err
			}
			return writeOut(cmd, opt, map[string]any{"extended": true, "result": out})
		},
	}
	cmd.Flags().IntVar(&sessionID, "session-id", 0, "Active parking session id")
	cmd.Flags().IntVar(&duration, "duration-minutes", 0, "Additional minutes")
	cmd.Flags().IntVar(&billingMethodID, "billing-method-id", 0, "Saved billing method id")
	cmd.Flags().IntVar(&timeBlockID, "timeblock-id", 0, "Optional time block id from zone info")
	cmd.Flags().BoolVar(&enableLive, "enable-live-parking", false, "Explicit opt-in to charge a payment method")
	cmd.Flags().BoolVar(&ownerApproved, "owner-approved", false, "Explicit owner approval")
	cmd.Flags().StringVar(&confirm, "confirm", "", "Must be exactly: "+extendConfirmPhrase)
	return cmd
}

func newSessionStopCmd(opt *Options) *cobra.Command {
	var sessionID int
	var enableLive, ownerApproved bool
	var confirm string
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop an active parking session early when zone allows (hard-gated)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if sessionID <= 0 {
				return exitcode.Usagef("--session-id is required")
			}
			if !enableLive || !ownerApproved || confirm != stopConfirmPhrase {
				return exitcode.Usagef("refusing live stop: require --enable-live-parking --owner-approved --confirm %q", stopConfirmPhrase)
			}
			c, err := opt.newClient()
			if err != nil {
				return err
			}
			if c.Session == nil || c.Session.CookieHeader() == "" {
				return exitcode.Authf("authenticated session required; run auth login")
			}
			sess, err := c.GetSession(strconv.Itoa(sessionID))
			if err != nil {
				return err
			}
			if !sess.CanStop {
				return exitcode.Usagef("session %d cannot be stopped early (can_stop=false)", sessionID)
			}
			out, err := c.StopSession(client.StopSessionInput{SessionID: sessionID})
			if err != nil {
				return err
			}
			return writeOut(cmd, opt, map[string]any{"stopped": true, "result": out})
		},
	}
	cmd.Flags().IntVar(&sessionID, "session-id", 0, "Active parking session id")
	cmd.Flags().BoolVar(&enableLive, "enable-live-parking", false, "Explicit opt-in to mutate remote session")
	cmd.Flags().BoolVar(&ownerApproved, "owner-approved", false, "Explicit owner approval")
	cmd.Flags().StringVar(&confirm, "confirm", "", "Must be exactly: "+stopConfirmPhrase)
	return cmd
}

func addStartFlags(cmd *cobra.Command, zone *string, duration, vehicleID, billingMethodID *int, space *string) {
	cmd.Flags().StringVar(zone, "zone", "", "Zone/signage code")
	cmd.Flags().IntVar(duration, "duration-minutes", 0, "Parking duration in minutes")
	cmd.Flags().IntVar(vehicleID, "vehicle-id", 0, "Saved vehicle id")
	cmd.Flags().IntVar(billingMethodID, "billing-method-id", 0, "Saved billing method id")
	cmd.Flags().StringVar(space, "space-number", "", "Space number when zone requires it")
}
