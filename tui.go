package main

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type TUI struct {
	header       *tview.TextView
	app          *tview.Application
	stdoutViewer *tview.TextView
	helpBar      *tview.TextView
}

// Set initial theme overrides, so tview uses default
// system colours rather than tcell theme overrides
func (tui *TUI) SetTheme() {
	tview.Styles.PrimitiveBackgroundColor = tcell.ColorDefault
	tview.Styles.ContrastBackgroundColor = tcell.ColorDefault
}

func NewHeader(tui *TUI) *tview.TextView {
	return tview.NewTextView().
		SetDynamicColors(true).
		SetText(HEADER_TEXT)
}

// TView application
func NewApplication() *tview.Application {
	return tview.NewApplication()
}

// Show command output text
func NewStdoutViewer(tui *TUI) *tview.TextView {
	return tview.NewTextView().
		SetDynamicColors(true).
		SetText(STDOUT_TEXT)
}

// Construct all UI components
func NewUI() *TUI {
	tui := TUI{}
	tui.SetTheme()

	tui.app = NewApplication()
	tui.header = NewHeader(&tui)
	tui.helpBar = NewHelpbar(&tui)
	tui.stdoutViewer = NewStdoutViewer(&tui)

	return &tui
}

// Arrange TUI components into a grid
func (tui *TUI) Grid() *tview.Grid {
	return tview.NewGrid().
		SetRows(1, 0, 1, 1).
		SetColumns(0).
		AddItem(tui.header, ROW_0, COL_0, ROWSPAN_1, COLSPAN_1, MINWIDTH_0, MINHEIGHT_0, true).
		AddItem(tui.stdoutViewer, ROW_1, COL_0, ROWSPAN_1, COLSPAN_1, MINWIDTH_0, MINHEIGHT_0, true).
		AddItem(tview.NewTextView(), ROW_2, COL_0, ROWSPAN_1, COLSPAN_1, MINWIDTH_0, MINHEIGHT_0, false).
		AddItem(tui.helpBar, ROW_3, COL_0, ROWSPAN_1, COLSPAN_1, MINWIDTH_0, MINHEIGHT_0, false)
}

// Start the TUI
func (tui *TUI) Start() {
	grid := tui.Grid()

	if err := tui.app.SetRoot(grid, true).SetFocus(grid).Run(); err != nil {
		fmt.Printf("RL: Application crashed! %v", err)
	}
}

// Show help-text to help user's use Replit
func NewHelpbar(tui *TUI) *tview.TextView {
	return tview.NewTextView().
		SetDynamicColors(true).
		SetText(HELP_TEXT)
}
