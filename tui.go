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

func NewApplication() *tview.Application {
	return tview.NewApplication()
}

func NewStdoutViewer(tui *TUI) *tview.TextView {
	return tview.NewTextView().
		SetDynamicColors(true).
		SetText("Waiting for program execution...")
}

func NewUI() *TUI {
	tui := TUI{}
	tui.SetTheme()

	tui.app = NewApplication()
	tui.header = NewHeader(&tui)
	tui.helpBar = NewHelpbar(&tui)
	tui.stdoutViewer = NewStdoutViewer(&tui)

	grid := tview.NewGrid().
		SetRows(1, 0, 1, 1).
		SetColumns(0).
		AddItem(tui.header, ROW_0, COL_0, ROWSPAN_1, COLSPAN_1, MINWIDTH_0, MINHEIGHT_0, true).
		AddItem(tui.stdoutViewer, ROW_1, COL_0, ROWSPAN_1, COLSPAN_1, MINWIDTH_0, MINHEIGHT_0, true).
		AddItem(tview.NewTextView(), ROW_2, COL_0, ROWSPAN_1, COLSPAN_1, MINWIDTH_0, MINHEIGHT_0, false).
		AddItem(tui.helpBar, ROW_3, COL_0, ROWSPAN_1, COLSPAN_1, MINWIDTH_0, MINHEIGHT_0, false)

	if err := tui.app.SetRoot(grid, true).SetFocus(grid).Run(); err != nil {
		fmt.Printf("RL: Application crashed! %v", err)
	}

	return &tui
}

func NewHelpbar(tui *TUI) *tview.TextView {
	return tview.NewTextView().
		SetDynamicColors(true).
		SetText(HELP_TEXT)
}
