package tui

import (
	"fmt"
	"testing"

	"github.com/rickihastings/spinner/internal/agent"
	"github.com/rickihastings/spinner/internal/provider"
	"github.com/stretchr/testify/assert"
)

// Note: Tests for event formatting have been moved to internal/agent/claude/formatter_test.go
// This file now only tests TUI-specific formatting helpers (metrics, bytes)

func TestMetricsFormatting_InitialState(t *testing.T) {
	// Test that empty metrics don't show invalid state
	m := provider.ContainerMetrics{}

	// When State is empty string (zero value), it should not display as "container"
	assert.NotEqual(t, "container", string(m.State), "Empty state should not be 'container'")
	assert.Equal(t, "", string(m.State), "Empty state should be empty string")
}

func TestMetricsFormatting_ValidStates(t *testing.T) {
	// Test all valid container states
	states := []provider.ContainerState{
		provider.StateRunning,
		provider.StateStopped,
		provider.StateExited,
		provider.StateUnknown,
	}

	for _, state := range states {
		t.Run(string(state), func(t *testing.T) {
			assert.NotEmpty(t, string(state), "State should not be empty")
			assert.NotEqual(t, "container", string(state), "State should not be 'container'")
		})
	}
}

func TestFormatMemoryValue(t *testing.T) {
	tests := []struct {
		bytes    uint64
		expected string
	}{
		{100 * 1024, "[cyan]100.00 KB[-]"},
		{512 * 1024 * 1024, "[cyan]512.00 MB[-]"},
		{2 * 1024 * 1024 * 1024, "[cyan]2.00 GB[-]"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatMemoryValue(tt.bytes)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWatchUI_TestMode(t *testing.T) {
	// Test that watch UI doesn't block when running in test mode
	logCh := make(chan agent.Event, 1)
	metricsCh := make(chan provider.ContainerMetrics, 1)

	// Create a watch UI instance with minimal context
	ui := NewWatchUI("test-container", &mockFormatter{}, WatchContext{})

	// Enable test mode to prevent TUI startup
	ui.setTestMode(true)

	// Close channels immediately to simulate end of data
	close(logCh)
	close(metricsCh)

	// This should not block in test mode
	err := ui.Run(logCh, metricsCh)

	// Verify it completed successfully
	assert.NoError(t, err)
}

func TestWatchUI_ScrollStateTransitions(t *testing.T) {
	ui := NewWatchUI("test-container", &mockFormatter{}, WatchContext{})

	// Initially auto-scroll is active (userScrolled = false)
	assert.False(t, ui.userScrolled, "userScrolled should be false initially")

	// Scrolling up should pause auto-scroll
	ui.setUserScrolled(true)
	assert.True(t, ui.userScrolled, "scrolling up should set userScrolled to true")

	// Pressing End should resume auto-scroll
	ui.setUserScrolled(false)
	assert.False(t, ui.userScrolled, "End key should clear userScrolled")

	// Scroll up again, then simulate reaching bottom
	ui.setUserScrolled(true)
	assert.True(t, ui.userScrolled)
	// Reaching bottom clears userScrolled
	ui.setUserScrolled(false)
	assert.False(t, ui.userScrolled, "reaching bottom should clear userScrolled")
}

func TestWatchUI_FooterTextContent(t *testing.T) {
	ui := NewWatchUI("test-container", &mockFormatter{}, WatchContext{})

	// Default footer (not scrolled)
	ui.updateFooter()
	footerText := ui.footer.GetText(false)
	assert.Contains(t, footerText, "scroll")
	assert.Contains(t, footerText, "quit")
	assert.NotContains(t, footerText, "SCROLLED")

	// When user has scrolled, footer should show SCROLLED indicator
	ui.setUserScrolled(true)
	footerText = ui.footer.GetText(false)
	assert.Contains(t, footerText, "SCROLLED")
	assert.Contains(t, footerText, "scroll")
	assert.Contains(t, footerText, "quit")

	// When user returns to bottom, SCROLLED indicator disappears
	ui.setUserScrolled(false)
	footerText = ui.footer.GetText(false)
	assert.NotContains(t, footerText, "SCROLLED")
}

func TestWatchUI_PageHeight(t *testing.T) {
	ui := NewWatchUI("test-container", &mockFormatter{}, WatchContext{})

	// pageHeight should return at least 1 even when logView has no dimensions
	height := ui.pageHeight()
	assert.GreaterOrEqual(t, height, 1, "pageHeight should be at least 1")
}

func TestWatchUI_HeaderToggle(t *testing.T) {
	// Default: header visible
	ui := NewWatchUI("test-container", &mockFormatter{}, WatchContext{HeaderVisible: true})
	assert.True(t, ui.headerVisible, "header should be visible initially when HeaderVisible=true")

	// Toggle header off
	ui.toggleHeader()
	assert.False(t, ui.headerVisible, "header should be hidden after toggle")

	// Toggle header back on
	ui.toggleHeader()
	assert.True(t, ui.headerVisible, "header should be visible after second toggle")
}

func TestWatchUI_HeaderStartHidden(t *testing.T) {
	// Start with header hidden via config
	ui := NewWatchUI("test-container", &mockFormatter{}, WatchContext{HeaderVisible: false})
	assert.False(t, ui.headerVisible, "header should be hidden when HeaderVisible=false")

	// Toggle header on
	ui.toggleHeader()
	assert.True(t, ui.headerVisible, "header should be visible after toggle")
}

func TestWatchUI_HelpOverlay_ShowAndDismiss(t *testing.T) {
	ui := NewWatchUI("test-container", &mockFormatter{}, WatchContext{HeaderVisible: true})

	// Initially help is not visible
	assert.False(t, ui.helpVisible, "helpVisible should be false initially")

	// Show help
	ui.showHelp()
	assert.True(t, ui.helpVisible, "helpVisible should be true after showHelp")

	// Dismiss help
	ui.hideHelp()
	assert.False(t, ui.helpVisible, "helpVisible should be false after hideHelp")
}

func TestWatchUI_HelpOverlay_ToggleCycle(t *testing.T) {
	ui := NewWatchUI("test-container", &mockFormatter{}, WatchContext{HeaderVisible: true})

	// Show → hide → show cycle
	ui.showHelp()
	assert.True(t, ui.helpVisible)

	ui.hideHelp()
	assert.False(t, ui.helpVisible)

	ui.showHelp()
	assert.True(t, ui.helpVisible)
}

func TestWatchUI_HelpOverlay_Content(t *testing.T) {
	ui := NewWatchUI("test-container", &mockFormatter{}, WatchContext{})

	// Help overlay should contain shortcut descriptions
	helpText := ui.helpOverlay.GetText(false)
	assert.Contains(t, helpText, "Scroll line")
	assert.Contains(t, helpText, "Scroll page")
	assert.Contains(t, helpText, "Top/Bottom")
	assert.Contains(t, helpText, "Toggle header")
	assert.Contains(t, helpText, "This help")
	assert.Contains(t, helpText, "Quit")
}

func TestWatchUI_PagesStructure(t *testing.T) {
	ui := NewWatchUI("test-container", &mockFormatter{}, WatchContext{HeaderVisible: true})

	// Pages should exist
	assert.NotNil(t, ui.pages, "pages should not be nil")

	// Help overlay should exist
	assert.NotNil(t, ui.helpOverlay, "helpOverlay should not be nil")
}

func TestWatchUI_RenderHeaderWide(t *testing.T) {
	ui := NewWatchUI("test-container", &mockFormatter{}, WatchContext{
		Branch:        "main",
		Agent:         "opus-4",
		Environment:   "production",
		ContainerID:   "abc123",
		ImageID:       "def456",
		MaxIterations: 100,
		HeaderVisible: true,
	})

	// Simulate metrics
	ui.mu.Lock()
	ui.lastMetrics = provider.ContainerMetrics{
		State:      provider.StateRunning,
		CPUPercent: 12.3,
		MemoryUsed: 1024 * 1024 * 512, // 512 MB
	}
	ui.currentIter = 5
	ui.mu.Unlock()

	// Render wide header directly
	ui.header.Clear()
	ui.renderHeaderWide(ui.lastMetrics, 120)
	text := ui.header.GetText(false)

	assert.Contains(t, text, "Branch:")
	assert.Contains(t, text, "main")
	assert.Contains(t, text, "Status:")
	assert.Contains(t, text, "running")
	assert.Contains(t, text, "Iterations:")
	assert.Contains(t, text, "5/100")
	assert.Contains(t, text, "Env:")
	assert.Contains(t, text, "Agent:")
	assert.Contains(t, text, "CPU:")
	assert.Contains(t, text, "Memory:")
}

func TestWatchUI_RenderHeaderCompact(t *testing.T) {
	ui := NewWatchUI("test-container", &mockFormatter{}, WatchContext{
		Branch:        "main",
		MaxIterations: 100,
		HeaderVisible: true,
	})

	ui.mu.Lock()
	ui.lastMetrics = provider.ContainerMetrics{
		State:      provider.StateRunning,
		CPUPercent: 12.3,
		MemoryUsed: 1024 * 1024 * 512,
	}
	ui.currentIter = 5
	ui.mu.Unlock()

	ui.header.Clear()
	ui.renderHeaderCompact(ui.lastMetrics, 60)
	text := ui.header.GetText(false)

	// Compact mode should contain essential fields
	assert.Contains(t, text, "running")
	assert.Contains(t, text, "5/100")
	assert.Contains(t, text, "main")
	assert.Contains(t, text, "12.3%")
	assert.Contains(t, text, "512.00 MB")

	// Compact mode should NOT contain full labels
	assert.NotContains(t, text, "Branch:")
	assert.NotContains(t, text, "Status:")
	assert.NotContains(t, text, "Env:")
	assert.NotContains(t, text, "Agent:")
}

func TestWatchUI_RenderHeaderMinimal(t *testing.T) {
	ui := NewWatchUI("test-container", &mockFormatter{}, WatchContext{
		Branch:        "main",
		MaxIterations: 100,
		HeaderVisible: true,
	})

	ui.mu.Lock()
	ui.lastMetrics = provider.ContainerMetrics{
		State:      provider.StateRunning,
		CPUPercent: 5.0,
		MemoryUsed: 1024 * 1024 * 256,
	}
	ui.currentIter = 3
	ui.mu.Unlock()

	ui.header.Clear()
	ui.renderHeaderMinimal(ui.lastMetrics)
	text := ui.header.GetText(false)

	// Minimal mode should show essential status
	assert.Contains(t, text, "running")
	assert.Contains(t, text, "3/100")
	assert.Contains(t, text, "5.0%")
	assert.Contains(t, text, "256.00 MB")

	// Minimal mode should NOT have separators
	assert.NotContains(t, text, "│")
}

func TestWatchUI_RenderHeaderDispatch(t *testing.T) {
	ui := NewWatchUI("test-container", &mockFormatter{}, WatchContext{
		Branch:        "main",
		MaxIterations: 50,
		HeaderVisible: true,
	})

	ui.mu.Lock()
	ui.lastMetrics = provider.ContainerMetrics{
		State:      provider.StateRunning,
		CPUPercent: 8.0,
		MemoryUsed: 1024 * 1024 * 128,
	}
	ui.currentIter = 10
	ui.mu.Unlock()

	// We can't directly test the dispatch since GetInnerRect returns (0,0,0,0) in test,
	// but we can verify the method doesn't panic
	ui.renderHeader()
}

func TestWatchUI_StatusBarStyle(t *testing.T) {
	ui := NewWatchUI("test-container", &mockFormatter{}, WatchContext{})

	// Footer should have dark background (status bar style)
	bg := ui.footer.GetBackgroundColor()
	assert.NotEqual(t, int32(0), int32(bg), "footer should have a non-default background color")

	// Footer text should be left-aligned and contain keyboard shortcuts
	footerText := ui.footer.GetText(false)
	assert.Contains(t, footerText, "quit")
	assert.Contains(t, footerText, "scroll")
	assert.Contains(t, footerText, "header")
	assert.Contains(t, footerText, "help")
}

func TestWatchUI_CompactBranchTruncation(t *testing.T) {
	ui := NewWatchUI("test-container", &mockFormatter{}, WatchContext{
		Branch: "very-long-feature-branch-name-that-should-be-truncated",
	})

	result := ui.compactBranch(40)
	// Should be truncated with "..." since 40/3 ≈ 13 chars max
	assert.Contains(t, result, "...")
	assert.Contains(t, result, "very-long")
}

func TestWatchUI_CompactStatusColors(t *testing.T) {
	ui := NewWatchUI("test-container", &mockFormatter{}, WatchContext{})

	tests := []struct {
		state         provider.ContainerState
		expectedColor string
		expectedText  string
	}{
		{provider.StateRunning, "green", "running"},
		{provider.StateStopped, "yellow", "stopped"},
		{provider.StateExited, "red", "exited"},
		{provider.StateUnknown, "gray", "unknown"},
		{"", "gray", "unknown"},
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			m := provider.ContainerMetrics{State: tt.state}
			result := ui.compactStatus(m)
			assert.Contains(t, result, tt.expectedColor)
			assert.Contains(t, result, tt.expectedText)
		})
	}
}

func TestWatchUI_CompactMemoryFormats(t *testing.T) {
	ui := NewWatchUI("test-container", &mockFormatter{}, WatchContext{})

	// Error case
	m := provider.ContainerMetrics{Error: fmt.Errorf("fail")}
	assert.Contains(t, ui.compactMemory(m), "ERR")

	// Memory used
	m = provider.ContainerMetrics{MemoryUsed: 1024 * 1024 * 512}
	assert.Contains(t, ui.compactMemory(m), "512.00 MB")

	// Memory percent
	m = provider.ContainerMetrics{MemoryPercent: 45.2}
	assert.Contains(t, ui.compactMemory(m), "45.2%")

	// N/A case
	m = provider.ContainerMetrics{}
	assert.Contains(t, ui.compactMemory(m), "N/A")
}

func TestWatchUI_CompactCPU(t *testing.T) {
	ui := NewWatchUI("test-container", &mockFormatter{}, WatchContext{})

	// Error case
	m := provider.ContainerMetrics{Error: fmt.Errorf("fail")}
	assert.Contains(t, ui.compactCPU(m), "ERR")

	// Normal case
	m = provider.ContainerMetrics{CPUPercent: 25.5}
	assert.Contains(t, ui.compactCPU(m), "25.5%")
}

// mockFormatter is a simple formatter for testing
type mockFormatter struct{}

func (m *mockFormatter) FormatEvent(_ *agent.Event) (string, bool) {
	return "", false
}
