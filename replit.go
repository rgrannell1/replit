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
func LaunchEditor(editorChan chan<- *exec.Cmd, file *EditorFile) {
	editor, _ := GetEditor()
	var cmd *exec.Cmd

	if editor == "code" {
		// having to change line-position is a little irritating
		cmd = exec.Command(editor, "--goto", file.File.Name()+":2")
	} else {
		cmd = exec.Command(editor, file.File.Name())
	}

	cmd.Start()
	editorChan <- cmd
}

type FileWatcher struct {
	Done  bool
	Files *[]string
}

func (watch *FileWatcher) Stop() {
	watch.Done = true
}

func (watch *FileWatcher) Stdin() *bytes.Buffer {
	byteStr := []byte(strings.Join(*watch.Files, "\n"))

	return bytes.NewBuffer(byteStr)
}

func (watch *FileWatcher) Start(tui *TUI) {
	go func() {
		for {
			if watch.Done {
				return
			}

			cmd := exec.Command("entr", "-zps", "echo 0")
			cmd.Stdin = watch.Stdin()
			cmd.Run()

			tui.actions.fileChange.Broadcast()
		}
	}()
}

// Observe file-changes
func ObserveFileChanges(args *ReplitArgs, tui *TUI) (FileWatcher, error) {
	targetFile := args.EditorFile
	dpath := args.Dpath

	var files *[]string

	if targetFile.IsTempFile {
		files = &[]string{targetFile.File.Name()}
	} else {
		var err error
		files, err = ListDirectory(dpath)

		if err != nil {
			return FileWatcher{}, err
		}
	}

	watch := FileWatcher{false, files}

	return watch, nil
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

type LanguageState struct {
	Time time.Time
	Lock sync.Mutex
	Cmd  *exec.Cmd
}

func RunLanguage(args *ReplitArgs, tui *TUI, state *LanguageState) {
	var cmd *exec.Cmd

	// kill the running process
	killProcess := func() {
		if cmd != nil {
			cmd.Process.Kill()
			cmd = nil
		}
	}

	// update stdout
	onFileChange := func() {
		now := time.Now()

		threshold := time.Second * 2
		if state.Time.Sub(now) > threshold {
			// too slow; even though it is running just kill the process and release the lock
			// so the new content can be run.

			if cmd != nil {
				cmd.Process.Kill()
				state.Lock.Unlock()
			}
		}

		state.Lock.Lock()

		// clear stdout
		stdoutViewer := tui.stdoutViewer
		stderrViewer := tui.stderrViewer

		stdoutViewer.Lock()
		stdoutViewer.Clear()
		stdoutViewer.Unlock()

		stderrViewer.Lock()
		stderrViewer.Clear()
		stderrViewer.Unlock()

		state.Time = now
		// call the language against a file
		cmd = exec.Command(args.Lang, args.EditorFile.File.Name())
		cmd.Stdout = stdoutViewer
		cmd.Stderr = stderrViewer

		startCommandTime := time.Now()
		cmd.Run()
		commandEnd := time.Since(startCommandTime)

		tui.UpdateRunTime(commandEnd)
		tui.UpdateRunCount()

		tui.app.Draw()
		state.Lock.Unlock()
	}

	// run on process-kill
	attachListener(tui.actions.killProcess, killProcess)

	// run on file-change
	attachListener(tui.actions.fileChange, onFileChange)
}

// Core application
func ReplIt(opts docopt.Opts) int {
	// read and validate arguments
	args, exitCode := ReadArgs(opts)
	if exitCode >= 0 {
		return exitCode
	}

	tui := NewUI(&args)

	tui.SetTheme()

	go func(tui *TUI) {
		tui.Start()
	}(tui)

	editorChan := make(chan *exec.Cmd)

	// launch an editor asyncronously
	go LaunchEditor(editorChan, args.EditorFile)

	// start entr; read the file (and optionally a directory) and live-reload
	state := LanguageState{
		time.Now(),
		sync.Mutex{},
		nil,
	}

	fileWatcher, err := ObserveFileChanges(&args, tui)
	if err != nil {
		panic(err)
	}

	go fileWatcher.Start(tui)

	go RunLanguage(&args, tui, &state)

	// Terminate program when an exit signal is received, and tidy up termporary files and processes

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	<-sigs
	close(sigs)

	var doneGroup sync.WaitGroup
	doneGroup.Add(2)

	// close each channel

	// kill editor
	go func() {
		defer doneGroup.Done()

		editor := <-editorChan
		editor.Process.Kill()
		close(editorChan)
	}()

	// remove temporary file
	go func() {
		defer doneGroup.Done()

		targetFile := args.EditorFile

		if targetFile.IsTempFile {
			name := targetFile.File.Name()
			os.Remove(name)
		}
	}()

	tui.app.Stop()
	doneGroup.Wait()

	return 0
}
