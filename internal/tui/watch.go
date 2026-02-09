package tui

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/gdamore/tcell/v2"
	"github.com/rickihastings/spinner/internal/agent"
	"github.com/rickihastings/spinner/internal/agent/claude"
	"github.com/rickihastings/spinner/internal/provider"
	"github.com/rivo/tview"
	"golang.org/x/term"
)

// WatchUI represents the TUI for watching container logs and metrics
type WatchUI struct {
	app         *tview.Application
	header      *tview.TextView
	logView     *tview.TextView
	footer      *tview.TextView
	layout      *tview.Flex
	ctx         context.Context
	cancel      context.CancelFunc
	mu          sync.Mutex
	lastMetrics provider.ContainerMetrics
	formatter   agent.EventFormatter

	// Execution context
	containerName string
	branch        string
	agent         string
	environment   string
	containerID   string
	imageID       string
	maxIterations int
	currentIter   int

	// Test mode flag - when true, skip TUI startup
	testMode bool
}

// WatchContext holds the execution context for the watch UI
type WatchContext struct {
	Branch        string
	Agent         string
	Environment   string
	ContainerID   string
	ImageID       string
	MaxIterations int
}

// isTestEnvironment detects if we're running in a test environment or without a terminal
func isTestEnvironment() bool {
	// Check if running under 'go test'
	if flag.Lookup("test.v") != nil {
		return true
	}

	// Check if stdout is not a terminal (headless environment)
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return true
	}

	return false
}

// NewWatchUI creates a new watch UI instance
func NewWatchUI(containerName string, formatter agent.EventFormatter, wctx WatchContext) *WatchUI {
	ctx, cancel := context.WithCancel(context.Background())

	app := tview.NewApplication()

	// Create header view for container metadata and metrics
	header := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	header.SetBorder(true).
		SetTitle(fmt.Sprintf(" Container: %s ", containerName)).
		SetBorderColor(tcell.ColorGray).
		SetBorderPadding(0, 0, 1, 1)

	// Create log view with auto-scrolling
	logView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true)
	logView.SetBorder(true).
		SetTitle(" Logs ").
		SetBorderColor(tcell.ColorGray).
		SetBorderPadding(0, 0, 1, 1)

	// Set up auto-scroll after logView is created
	logView.SetChangedFunc(func() {
		app.QueueUpdateDraw(func() {
			logView.ScrollToEnd()
		})
	})

	// Create footer for help text
	footer := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignRight)
	footer.SetText("[darkgray]Press q to quit[-]")

	// Create split-pane layout
	layout := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(header, 5, 0, false).
		AddItem(logView, 0, 1, true).
		AddItem(footer, 1, 0, false)

	ui := &WatchUI{
		app:           app,
		header:        header,
		logView:       logView,
		footer:        footer,
		layout:        layout,
		ctx:           ctx,
		cancel:        cancel,
		formatter:     formatter,
		containerName: containerName,
		branch:        wctx.Branch,
		agent:         wctx.Agent,
		environment:   wctx.Environment,
		containerID:   wctx.ContainerID,
		imageID:       wctx.ImageID,
		maxIterations: wctx.MaxIterations,
		currentIter:   0,
		testMode:      isTestEnvironment(), // Auto-detect test mode
	}

	// Set up keyboard handlers
	ui.setupKeyboardHandlers()

	return ui
}

// setupKeyboardHandlers configures keyboard input handling
func (ui *WatchUI) setupKeyboardHandlers() {
	ui.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Handle 'q' key to quit
		if event.Rune() == 'q' || event.Rune() == 'Q' {
			ui.Stop()
			return nil
		}
		// Handle Ctrl+C to quit
		if event.Key() == tcell.KeyCtrlC {
			ui.Stop()
			return nil
		}

		return event
	})
}

// Run starts the TUI and begins consuming from log and metrics channels
func (ui *WatchUI) Run(logCh <-chan agent.Event, metricsCh <-chan provider.ContainerMetrics) error {
	// If in test mode, don't start TUI
	if ui.testMode {
		return ui.runInTestMode(logCh, metricsCh)
	}

	// Start goroutines to consume from channels
	var wg sync.WaitGroup

	// Log consumer
	wg.Add(1)

	go func() {
		defer wg.Done()

		ui.consumeLogs(logCh)
	}()

	// Metrics consumer
	wg.Add(1)

	go func() {
		defer wg.Done()

		ui.consumeMetrics(metricsCh)
	}()

	// Start the application
	ui.app.SetRoot(ui.layout, true)

	// Run in goroutine to allow cleanup
	errCh := make(chan error, 1)

	go func() {
		if err := ui.app.Run(); err != nil {
			errCh <- err
		}
	}()

	// Wait for either app error or context cancellation
	select {
	case err := <-errCh:
		ui.cancel()
		wg.Wait()

		return err
	case <-ui.ctx.Done():
		wg.Wait()

		return nil
	}
}

// Stop gracefully shuts down the TUI
func (ui *WatchUI) Stop() {
	ui.app.Stop()
	ui.cancel()

	// Clear screen and reset cursor position
	fmt.Print("\033[2J\033[H\033[?25h")
}

// consumeLogs reads log entries from the channel and updates the log view
func (ui *WatchUI) consumeLogs(logCh <-chan agent.Event) {
	for {
		select {
		case event, ok := <-logCh:
			if !ok {
				return
			}

			ui.appendLog(event)
		case <-ui.ctx.Done():
			return
		}
	}
}

// consumeMetrics reads metrics from the channel and updates the header
func (ui *WatchUI) consumeMetrics(metricsCh <-chan provider.ContainerMetrics) {
	for {
		select {
		case m, ok := <-metricsCh:
			if !ok {
				return
			}

			ui.updateMetrics(m)
		case <-ui.ctx.Done():
			return
		}
	}
}

// appendLog adds a log entry to the log view
func (ui *WatchUI) appendLog(event agent.Event) {
	// Extract system init data to populate missing context
	if event.Type == claude.EventTypeSystemInit {
		if initData, ok := event.Data.(claude.SystemInitData); ok {
			ui.mu.Lock()
			// Update agent if not already set
			if ui.agent == "" && initData.Model != "" {
				ui.agent = initData.Model
			}

			ui.mu.Unlock()
			// Trigger header update
			ui.app.QueueUpdateDraw(func() {
				ui.renderHeader()
			})
		}
	}

	ui.app.QueueUpdateDraw(func() {
		var line string

		// Format the event using the formatter
		if formatted, ok := ui.formatter.FormatEvent(&event); ok {
			line = formatted + "\n"
		} else if event.Raw != "" {
			// Display raw message for unrecognized events
			line = fmt.Sprintf("[-]%s\n", event.Raw)
		} else {
			// Skip events we can't format
			return
		}

		_, _ = fmt.Fprint(ui.logView, line)
	})
}

// updateMetrics updates the header with current metrics
func (ui *WatchUI) updateMetrics(m provider.ContainerMetrics) {
	ui.mu.Lock()
	ui.lastMetrics = m

	if m.Iteration > 0 {
		ui.currentIter = m.Iteration
	}

	ui.mu.Unlock()

	ui.app.QueueUpdateDraw(func() {
		ui.renderHeader()
	})
}

// renderHeader renders the header with container metadata and metrics
func (ui *WatchUI) renderHeader() {
	ui.mu.Lock()
	m := ui.lastMetrics
	ui.mu.Unlock()

	ui.header.Clear()

	// Get terminal width and calculate column width
	_, _, width, _ := ui.header.GetInnerRect()
	// Account for separators: " │ " (3 chars each, 2 separators = 6 chars)
	// Border padding (left/right = 2 chars) and borders (2 chars) = 4 chars
	availableWidth := width - 10
	if availableWidth < 60 {
		availableWidth = 60 // Minimum width
	}

	colWidth := availableWidth / 3

	// Build rows using padded columns
	// Column 1: Branch, Status, Iterations
	// Column 2: Env, Memory, CPU
	// Column 3: Agent, Container, Image
	row1 := fmt.Sprintf("%s │ %s │ %s",
		padString(ui.formatBranchLine(colWidth), colWidth),
		padString(ui.formatEnvLine(colWidth), colWidth),
		padString(ui.formatAgentLine(colWidth), colWidth))
	row2 := fmt.Sprintf("%s │ %s │ %s",
		padString(ui.formatStatusLine(m), colWidth),
		padString(ui.formatMemoryLine(m), colWidth),
		padString(ui.formatContainerLine(colWidth), colWidth))
	row3 := fmt.Sprintf("%s │ %s │ %s",
		padString(ui.formatIterationsLine(), colWidth),
		padString(ui.formatCPULine(m), colWidth),
		padString(ui.formatImageLine(colWidth), colWidth))

	content := fmt.Sprintf("%s\n%s\n%s", row1, row2, row3)

	_, _ = fmt.Fprint(ui.header, content)
}

// padString pads a string to the specified width, accounting for tview color tags
func padString(s string, width int) string {
	// Strip color tags to calculate visible length
	visibleLen := visibleLength(s)

	if visibleLen >= width {
		return s
	}

	// Add padding spaces to reach desired width
	padding := width - visibleLen

	return s + strings.Repeat(" ", padding)
}

// visibleLength calculates the visible length of a string, excluding tview color tags
func visibleLength(s string) int {
	length := 0
	inTag := false

	for _, ch := range s {
		if ch == '[' {
			inTag = true
		} else if ch == ']' && inTag {
			inTag = false
		} else if !inTag {
			length++
		}
	}

	return length
}

// truncateForColumn truncates a value to fit within a column, accounting for label and color tags.
// label is the field label (e.g., "Branch:     "), value is the text to display.
// If appendEllipsis is true, adds "..." when truncating.
func truncateForColumn(value string, colWidth int, label string, appendEllipsis bool) string {
	if value == "" || value == "-" {
		return value
	}

	// Calculate available space: colWidth - label length - "[cyan]" (6) - "[-]" (4)
	labelLen := len(label)
	colorTagsLen := 10 // "[cyan]" + "[-]"
	maxLen := colWidth - labelLen - colorTagsLen

	// Ensure minimum space
	if maxLen < 3 {
		maxLen = 3
	}

	if len(value) > maxLen {
		if appendEllipsis && maxLen > 3 {
			return value[:maxLen-3] + "..."
		}

		return value[:maxLen]
	}

	return value
}

// formatStatusLine formats the status field
func (ui *WatchUI) formatStatusLine(m provider.ContainerMetrics) string {
	var stateColor string

	stateText := string(m.State)

	switch m.State {
	case provider.StateStopped:
		stateColor = "yellow"
	case provider.StateExited:
		stateColor = "red"
	case provider.StateUnknown:
		stateColor = "gray"
	case provider.StateRunning:
		stateColor = "green"
	default:
		stateColor = "white"
		stateText = "unknown"
	}

	return fmt.Sprintf("Status:     [%s]%s[-]", stateColor, stateText)
}

// formatBranchLine formats the branch field
func (ui *WatchUI) formatBranchLine(colWidth int) string {
	branch := ui.branch
	if branch == "" {
		branch = "-"
	}

	branch = truncateForColumn(branch, colWidth, "Branch:     ", true)

	return fmt.Sprintf("Branch:     [cyan]%s[-]", branch)
}

// formatAgentLine formats the agent field
func (ui *WatchUI) formatAgentLine(colWidth int) string {
	agentName := ui.agent
	if agentName == "" {
		agentName = "-"
	}

	agentName = truncateForColumn(agentName, colWidth, "Agent:      ", true)

	return fmt.Sprintf("Agent:      [cyan]%s[-]", agentName)
}

// formatIterationsLine formats the iterations field
func (ui *WatchUI) formatIterationsLine() string {
	if ui.maxIterations == 0 {
		return "Iterations: [cyan]-[-]"
	}

	return fmt.Sprintf("Iterations: [cyan]%d/%d[-]", ui.currentIter, ui.maxIterations)
}

// formatEnvLine formats the environment field
func (ui *WatchUI) formatEnvLine(colWidth int) string {
	env := ui.environment
	if env == "" {
		env = "-"
	}

	env = truncateForColumn(env, colWidth, "Env:        ", true)

	return fmt.Sprintf("Env:        [cyan]%s[-]", env)
}

// formatContainerLine formats the container ID field
func (ui *WatchUI) formatContainerLine(colWidth int) string {
	containerID := ui.containerID
	if containerID == "" {
		containerID = "-"
	}

	containerID = truncateForColumn(containerID, colWidth, "Container:  ", false)

	return fmt.Sprintf("Container:  [cyan]%s[-]", containerID)
}

// formatCPULine formats the CPU field
func (ui *WatchUI) formatCPULine(m provider.ContainerMetrics) string {
	var cpuStr string
	if m.Error != nil {
		cpuStr = "[red]Error[-]"
	} else {
		cpuStr = fmt.Sprintf("[cyan]%.1f%%[-]", m.CPUPercent)
	}

	return fmt.Sprintf("CPU:        %s", cpuStr)
}

// formatMemoryLine formats the memory field
func (ui *WatchUI) formatMemoryLine(m provider.ContainerMetrics) string {
	var memStr string
	if m.Error != nil {
		memStr = "[red]Error[-]"
	} else if m.MemoryUsed > 0 {
		// Docker-style: show absolute memory usage
		memStr = formatMemoryValue(m.MemoryUsed)
	} else if m.MemoryPercent > 0 {
		// GCP-style: show memory percentage (when Ops Agent is installed)
		memStr = fmt.Sprintf("[cyan]%.1f%%[-]", m.MemoryPercent)
	} else {
		// Memory metrics not available (e.g., GCP VMs without Ops Agent)
		memStr = "[darkgray]N/A[-]"
	}

	return fmt.Sprintf("Memory:     %s", memStr)
}

// formatImageLine formats the image ID field
func (ui *WatchUI) formatImageLine(colWidth int) string {
	imageID := ui.imageID
	if imageID == "" {
		imageID = "-"
	}

	imageID = truncateForColumn(imageID, colWidth, "Image:      ", false)

	return fmt.Sprintf("Image:      [cyan]%s[-]", imageID)
}

// formatMemoryValue formats memory usage in a short form
func formatMemoryValue(used uint64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)

	switch {
	case used >= GB:
		return fmt.Sprintf("[cyan]%.2f GB[-]", float64(used)/float64(GB))
	case used >= MB:
		return fmt.Sprintf("[cyan]%.2f MB[-]", float64(used)/float64(MB))
	default:
		return fmt.Sprintf("[cyan]%.2f KB[-]", float64(used)/float64(KB))
	}
}

// setTestMode enables test mode, which skips TUI startup.
// This should only be called from tests within this package.
func (ui *WatchUI) setTestMode(enabled bool) {
	ui.testMode = enabled
}

// runInTestMode runs watch in test mode without starting the TUI
func (ui *WatchUI) runInTestMode(logCh <-chan agent.Event, metricsCh <-chan provider.ContainerMetrics) error {
	// In test mode, just consume channels until they close or context is done
	var wg sync.WaitGroup

	wg.Add(2)

	// Consume logs
	go func() {
		defer wg.Done()

		for {
			select {
			case _, ok := <-logCh:
				if !ok {
					return
				}
			case <-ui.ctx.Done():
				return
			}
		}
	}()

	// Consume metrics
	go func() {
		defer wg.Done()

		for {
			select {
			case _, ok := <-metricsCh:
				if !ok {
					return
				}
			case <-ui.ctx.Done():
				return
			}
		}
	}()

	// Wait for both channels to close or context to be cancelled
	wg.Wait()

	return nil
}

// Context returns the UI's context for coordinating shutdown
func (ui *WatchUI) Context() context.Context {
	return ui.ctx
}
