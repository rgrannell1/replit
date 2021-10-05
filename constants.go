package main

const Usage = `
Usage:
	replit <lang>
	replit [-d <dir>|--directory <dir>] <lang> [<file>]

Description:
	replit launches

Environmental Variables:
	$VISUAL    The visual-code editor.

Arguments:
	<lang>    a language executable (e.g python3, node) to launch an interactive runner.
	<file>    optional. If selected, entr will run against this file.

Options:
  -d <dir>, --directory <dir>    the directory to monitor for changes
	`

const COMMAND_AND_LINE_ROWS = 2
const STDOUT_ROWS = 0
const SPACE_ROWS = 1
const HELP_ROWS = 1
const COMMAND_ROWS = 1

const ROW_0 = 0
const ROW_1 = 1
const ROW_2 = 2
const ROW_3 = 3
const ROW_4 = 4

const COL_0 = 0
const COL_1 = 1
const COL_2 = 2

const ROWSPAN_1 = 1
const COLSPAN_1 = 1
const COLSPAN_2 = 2
const COLSPAN_3 = 3

const MINHEIGHT_0 = 0

const MINWIDTH_0 = 0
const MINWIDTH_1 = 1

const FOCUS = true
const DONT_FOCUS = false

const HELP_TEXT = "Help"
const HEADER_TEXT = "[red]Replit[reset]"
const STDOUT_TEXT = "Waiting for program execution...\n"
