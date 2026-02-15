package cmd

import (
	"context"
	"fmt"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/rickihastings/spinner/internal/provider"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// listCmd is the production list command using the default provider factory.
var listCmd = NewListCommand(defaultFactory)

func init() {
	rootCmd.AddCommand(listCmd)
}

// NewListCommand creates a new list command with the given Factory.
// This constructor enables dependency injection for testing.
func NewListCommand(f *provider.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all spinner-managed instances across backends",
		Long: `List all spinner-managed instances across backends

USAGE:
  spinner list [--project <project> --zone <zone> --state-bucket <bucket>]

DESCRIPTION:
  The list command discovers and displays all spinner-managed instances
  across all configured backends (Docker, GCP). It shows instance status,
  execution state, iteration progress, and timing information.

  Docker is always queried. GCP is queried only when project/zone
  configuration is available (from flags, env vars, or .spinner.json).
  Individual backend errors are shown as warnings, not fatal errors.

  Only instances created with the spinner-managed=true label are shown.
  Containers created before label support was added will not appear.

COLUMNS:
  BACKEND      - Backend provider (docker, gcp)
  NAME         - Instance name
  STATUS       - Lifecycle status (running, stopped)
  STATE        - Agent execution state (running, completed, rate_limited, error)
  ITER         - Current/max iteration count
  AGE          - Time since instance started
  LAST UPDATE  - Time since last state update (⚠ stale if >2h for running)

EXAMPLES:
  # List all Docker instances
  spinner list

  # List Docker and GCP instances
  spinner list --project my-proj --zone us-central1-a --state-bucket my-bucket`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd, f)
		},
	}

	// GCP backend flags (optional — GCP is skipped if not configured)
	cmd.Flags().String(flagProject, "", "GCP project ID (enables GCP backend)")
	cmd.Flags().String(flagZone, "", "GCP zone (enables GCP backend)")
	cmd.Flags().String(flagStateBucket, "", "GCS bucket for state persistence (GCP backend)")

	return cmd
}

// staleThreshold is the duration after which a running instance with no
// state update is considered stale.
const staleThreshold = 2 * time.Hour

// runList executes the list command logic: query all backends and render results.
func runList(cmd *cobra.Command, f *provider.Factory) error {
	bindGCPFlags(cmd)

	ctx := context.Background()

	var allInstances []provider.InstanceInfo

	for _, backend := range f.Available() {
		// For GCP, skip silently if not configured
		if backend == provider.BackendGCP {
			if !gcpConfigured() {
				continue
			}
		}

		p, err := f.Create(backend)
		if err != nil {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "⚠ %s: %s\n", backend, err.Error())
			continue
		}

		instances, err := p.List(ctx)
		if err != nil {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "⚠ %s: %s\n", backend, err.Error())
			continue
		}

		allInstances = append(allInstances, instances...)
	}

	if len(allInstances) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No instances found.")
		return nil
	}

	// Sort: by backend, then running instances first, then by name
	sort.Slice(allInstances, func(i, j int) bool {
		if allInstances[i].Backend != allInstances[j].Backend {
			return allInstances[i].Backend < allInstances[j].Backend
		}

		if allInstances[i].Status != allInstances[j].Status {
			return allInstances[i].Status == provider.InstanceStatusRunning
		}

		return allInstances[i].Name < allInstances[j].Name
	})

	renderTable(cmd, allInstances)

	return nil
}

// gcpConfigured returns true if GCP project and zone are available from any source.
func gcpConfigured() bool {
	return viper.GetString(flagProject) != "" && viper.GetString(flagZone) != ""
}

// renderTable writes instances as a formatted table to the command's output.
func renderTable(cmd *cobra.Command, instances []provider.InstanceInfo) {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)

	_, _ = fmt.Fprintln(w, "BACKEND\tNAME\tSTATUS\tSTATE\tITER\tAGE\tLAST UPDATE")

	now := time.Now()

	for _, inst := range instances {
		iter := formatIter(inst.Iteration, inst.MaxIterations)
		age := formatDuration(inst.StartedAt, now)
		lastUpdate := formatDuration(inst.LastUpdated, now)

		// Add stale warning for running instances with old state updates
		if inst.Status == provider.InstanceStatusRunning && inst.LastUpdated != nil {
			if now.Sub(*inst.LastUpdated) > staleThreshold {
				lastUpdate += "  ⚠ stale"
			}
		}

		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			inst.Backend,
			inst.Name,
			string(inst.Status),
			formatState(inst.AgentStatus),
			iter,
			age,
			lastUpdate,
		)
	}

	_ = w.Flush()
}

// formatIter formats the iteration counter as "current/max" or "-" if unknown.
func formatIter(current, max int) string {
	if current == 0 && max == 0 {
		return "-"
	}

	if max == 0 {
		return fmt.Sprintf("%d", current)
	}

	return fmt.Sprintf("%d/%d", current, max)
}

// formatState returns the agent status or "-" if empty.
func formatState(status string) string {
	if status == "" {
		return "-"
	}

	return status
}

// formatDuration returns a human-readable relative time or "-" if nil.
func formatDuration(t *time.Time, now time.Time) string {
	if t == nil {
		return "-"
	}

	d := now.Sub(*t)
	if d < 0 {
		d = 0
	}

	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		days := int(d.Hours() / 24)
		return fmt.Sprintf("%dd ago", days)
	}
}
