package main

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type TUI struct {
	header           *tview.TextView
	app              *tview.Application
	stdoutViewer     *tview.TextView
	stderrViewer     *tview.TextView
	helpBar          *tview.TextView
	runCountViewer   *tview.TextView
	runSecondsViewer *tview.TextView
	runCount         int64
	runTime          int64
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
	return tview.NewApplication().EnableMouse(true)
}

// Show command output text
func NewStdoutViewer(tui *TUI) *tview.TextView {
	view := tview.NewTextView().
		SetDynamicColors(true)

	view.
		SetText(STDOUT_TEXT).Box.SetBorder(true)

	return view
}

// Show command output text
func NewStderrViewer(tui *TUI) *tview.TextView {
	view := tview.NewTextView().
		SetDynamicColors(true)

	view.
		SetText(STDERR_TEXT).Box.SetBorder(true)

	return view
}

func NewRunCount(tui *TUI) *tview.TextView {
	return tview.NewTextView().
		SetDynamicColors(true).
		SetText("run " + fmt.Sprint(tui.runCount) + " times")
}

func NewRunTime(tui *TUI) *tview.TextView {
	return tview.NewTextView().
		SetDynamicColors(true).
		SetText(fmt.Sprint(tui.runTime) + "ms")
}

// Construct all UI components
func NewUI(args *ReplitArgs) *TUI {
	tui := TUI{}
	tui.SetTheme()

	tui.app = NewApplication()
	tui.header = NewHeader(&tui)
	tui.helpBar = NewHelpbar(&tui, args)
	tui.stdoutViewer = NewStdoutViewer(&tui)
	tui.stderrViewer = NewStderrViewer(&tui)
	tui.runCountViewer = NewRunCount(&tui)
	tui.runSecondsViewer = NewRunTime(&tui)

	return &tui
}

func (tui *TUI) UpdateRunCount() {
	tui.runCount += 1
	tui.runCountViewer.SetText("run " + fmt.Sprint(tui.runCount) + " times")
}

func (tui *TUI) UpdateRunTime(diff time.Duration) {
	tui.runTime = diff.Milliseconds()
	tui.runSecondsViewer.SetText(fmt.Sprint(tui.runTime) + "ms")
}

// Arrange TUI components into a grid
func (tui *TUI) Grid() *tview.Grid {
	return tview.NewGrid().
		SetBorders(false).
		SetRows(1, 0, 1, 1).
		SetColumns(-4, -2, -1, -1).
		AddItem(tui.header, ROW_0, COL_0, ROWSPAN_1, COLSPAN_2, MINWIDTH_0, MINHEIGHT_0, true).
		AddItem(tui.runCountViewer, ROW_0, COL_2, ROWSPAN_1, COLSPAN_1, MINWIDTH_0, MINHEIGHT_0, true).
		AddItem(tui.runSecondsViewer, ROW_0, COL_3, ROWSPAN_1, COLSPAN_1, MINWIDTH_0, MINHEIGHT_0, true).
		AddItem(tui.stdoutViewer, ROW_1, COL_0, ROWSPAN_1, COLSPAN_1, MINWIDTH_0, MINHEIGHT_0, true).
		AddItem(tui.stderrViewer, ROW_1, COL_1, ROWSPAN_1, COLSPAN_3, MINWIDTH_0, MINHEIGHT_0, true).
		AddItem(tview.NewTextView(), ROW_2, COL_0, ROWSPAN_1, COLSPAN_4, MINWIDTH_0, MINHEIGHT_0, false).
		AddItem(tui.helpBar, ROW_3, COL_0, ROWSPAN_1, COLSPAN_4, MINWIDTH_0, MINHEIGHT_0, false)
}

// Start the TUI
func (tui *TUI) Start() {
	grid := tui.Grid()

	if err := tui.app.SetRoot(grid, true).SetFocus(grid).Run(); err != nil {
		fmt.Printf("RL: Application crashed! %v", err)
	}
}

// Show help-text to help user's use Replit
func NewHelpbar(tui *TUI, args *ReplitArgs) *tview.TextView {
	return tview.NewTextView().
		SetDynamicColors(true).
		SetText("Edit [red]" + args.EditorFile.File.Name() + "[reset] & save to run with [red]" + args.Lang + "[reset]")
}
