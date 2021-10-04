package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/docopt/docopt-go"
	"github.com/rivo/tview"
)

type EditorFile struct {
	IsTempFile bool
	File       *os.File
}

type ReplitArgs struct {
	EditorFile *EditorFile
	Dpath      string
	Lang       string
}

// List all files in directory
func ListDirectory(dir string) (*[]string, error) {
	dirInfo, err := os.Stat(dir)
	if err != nil {
		return nil, err
	}

	if !dirInfo.IsDir() {
		return nil, errors.New(dir + " was not a directory.")
	}

	files := []string{}

	// walk through directory and append files to a slice.
	filepath.Walk(dir, func(fpath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			files = append(files, fpath)
		}

		return nil
	})

	return &files, nil
}

// Launch an entr process and intercept output.
func LaunchEntr(args *ReplitArgs, tui *TUI, files *[]string) *exec.Cmd {
	watched := []byte(strings.Join(*files, "\n"))

	cmd := exec.Command("entr", "-c", "-s", args.Lang+" '"+args.EditorFile.File.Name()+"'")

	cmd.Stdin = bytes.NewBuffer(watched)
	cmd.Stdout = tview.ANSIWriter(tui.stdoutViewer)
	cmd.Stderr = tview.ANSIWriter(tui.stdoutViewer)

	go func() {
		for {
			time.Sleep(100 * time.Millisecond)
			tui.app.Draw()
		}
	}()

	cmd.Start()
	return cmd
}

// Check the requested language
func ValidateLanguage(language string) error {
	if !CommandExists(language) {
		return fmt.Errorf("language %s is not in PATH", language)
	}

	return nil
}

// Create and open a temporary file
func TargetFile(file string, lang string) (*EditorFile, error) {
	if len(file) == 0 {
		tgt, err := ioutil.TempFile("/tmp", "replit")
		if err != nil {
			return nil, err
		}

		tgt.WriteString("#!/usr/bin/env " + lang + "\n")

		return &EditorFile{
			true,
			tgt,
		}, nil
	}

	conn, err := os.Open(file)
	if err != nil {
		return nil, err
	}

	return &EditorFile{
		false,
		conn,
	}, nil
}

// Check whether a command exists
func CommandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// Get the user's preferred visual editor
func GetEditor() (string, error) {
	visual := os.Getenv("VISUAL")
	editor := ""

	if len(visual) > 0 {
		editor = visual
	} else {
		editor = "code"
	}

	if !CommandExists(editor) {
		return "", errors.New("the command '" + editor + "' is not in PATH; is it installed and available as a command?")
	}

	return editor, nil
}

// Launch the user's visual-editor, falling back to VSCode as a default.
func LaunchEditor(editorChan chan *exec.Cmd, file *EditorFile) {
	editor, _ := GetEditor()
	cmd := exec.Command(editor, file.File.Name())

	cmd.Start()
	editorChan <- cmd
}

// Start entr
func StartEntr(args *ReplitArgs, tui *TUI, entrChan chan *exec.Cmd) error {
	targetFile := args.EditorFile
	dpath := args.Dpath

	if targetFile.IsTempFile {
		// launch entr against a temporary file; only watch a single file

		go func(cmdChan chan *exec.Cmd) {
			files := &[]string{targetFile.File.Name()}
			cmdChan <- LaunchEntr(args, tui, files)
		}(entrChan)
	} else {
		// launch entr for a provided directory and file

		files, err := ListDirectory(dpath)

		if err != nil {
			return err
		}

		go func(cmdChan chan *exec.Cmd) {
			cmdChan <- LaunchEntr(args, tui, files)
		}(entrChan)
	}

	return nil
}

// Stop replit; terminate entr, editor, and remove the temporary file
func StopReplit(entrChan chan *exec.Cmd, editorChan chan *exec.Cmd, targetFile *EditorFile) {
	var doneGroup sync.WaitGroup
	doneGroup.Add(3)

	// kill entr command
	go func() {
		defer doneGroup.Done()

		entr := <-entrChan
		entr.Process.Kill()
	}()

	// kill editor
	go func() {
		defer doneGroup.Done()

		editor := <-editorChan
		editor.Process.Kill()
	}()

	// remove temporary file
	go func() {
		defer doneGroup.Done()

		if targetFile.IsTempFile {
			name := targetFile.File.Name()
			os.Remove(name)
		}
	}()

	doneGroup.Wait()
}

// Teriminate program when an exit signal is received, and tidy up termporary files and processes
func ExitHandler(entrChan chan *exec.Cmd, editorChan chan *exec.Cmd, targetFile *EditorFile) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	done := make(chan bool, 1)
	go func(done chan bool) {
		<-sigs
		done <- true
	}(done)

	<-done

	StopReplit(entrChan, editorChan, targetFile)
	os.Exit(0)
}

// Read docopt arguments and return parsed, provided parameters
func ReadArgs(opts docopt.Opts) (ReplitArgs, int) {
	dir, _ := opts.String("--directory")
	if len(dir) == 0 {
		dir, _ = os.Getwd()
	}

	dpath, err := filepath.Abs(dir)
	if err != nil {
		println("replit: failed to resolve directory path")
		return ReplitArgs{}, 1
	}

	lang, err := opts.String("<lang>")
	if err != nil {
		println("replit: could not read language")
		return ReplitArgs{}, 1
	}

	// check the editor is present; ignore the value for the moment
	_, err = GetEditor()

	if err != nil {
		panic(err)
	}

	langErr := ValidateLanguage(lang)
	if langErr != nil {
		panic(langErr)
	}

	file, _ := opts.String("<file>")
	targetFile, err := TargetFile(file, lang)

	if err != nil {
		panic(err)
	}

	return ReplitArgs{
		targetFile,
		dpath,
		lang,
	}, -1
}

// Core application
func ReplIt(opts docopt.Opts) int {
	tui := NewUI()
	tui.SetTheme()

	go func(tui *TUI) {
		tui.Start()
	}(tui)

	// read and validate arguments
	args, exitCode := ReadArgs(opts)
	if exitCode >= 0 {
		return exitCode
	}

	editorChan := make(chan *exec.Cmd)

	// launch an editor asyncronously
	go LaunchEditor(editorChan, args.EditorFile)

	// start entr; read the file (and optionally a directory) and live-reload
	entrChan := make(chan *exec.Cmd)
	go StartEntr(&args, tui, entrChan)

	// Teriminate program when an exit signal is received, and tidy up termporary files and processes
	ExitHandler(entrChan, editorChan, args.EditorFile)
	return 0
}
