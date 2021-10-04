package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/docopt/docopt-go"
)

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
func LaunchEntr(language string, targetFile *os.File, files *[]string) *exec.Cmd {
	watched := []byte(strings.Join(*files, "\n"))

	cmd := exec.Command("entr", "-s", language+" '"+targetFile.Name()+"'")

	var outputBuffer bytes.Buffer

	cmd.Stdin = bytes.NewBuffer(watched)
	cmd.Stdout = &outputBuffer
	cmd.Stderr = &outputBuffer

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
func TargetFile(file string, lang string) (*os.File, error) {
	if len(file) == 0 {
		tgt, err := ioutil.TempFile("/tmp", "replit")
		if err != nil {
			return nil, err
		}

		tgt.WriteString("#!/usr/bin/env " + lang + "\n")

		return tgt, nil
	}

	return os.Open(file)
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
func LaunchEditor(file *os.File, editorChan chan *exec.Cmd) {
	editor, _ := GetEditor()
	cmd := exec.Command(editor, file.Name())

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Start()
	editorChan <- cmd
}

func StartEntr(file string, lang string, targetFile *os.File, dpath string, entrChan chan *exec.Cmd) error {
	isTmpFile := len(file) == 0

	if isTmpFile {
		// launch entr against a temporary file
		go func(cmdChan chan *exec.Cmd) {
			cmdChan <- LaunchEntr(lang, targetFile, &[]string{targetFile.Name()})
		}(entrChan)
	} else {
		files, err := ListDirectory(dpath)

		if err != nil {
			return err
		}

		go func(cmdChan chan *exec.Cmd) {
			cmdChan <- LaunchEntr(lang, targetFile, files)
		}(entrChan)
	}

	return nil
}

func StopReplit(entrChan chan *exec.Cmd, editorChan chan *exec.Cmd, isTmpFile bool, targetFile *os.File) {
	var doneGroup sync.WaitGroup
	doneGroup.Add(3)

	go func() {
		defer doneGroup.Done()

		// kill entr command
		entr := <-entrChan
		entr.Process.Kill()
	}()

	go func() {
		defer doneGroup.Done()

		// kill editor
		editor := <-editorChan
		editor.Process.Kill()
	}()

	go func() {
		// remove temporary file
		defer doneGroup.Done()

		if isTmpFile {
			name := targetFile.Name()
			os.Remove(name)
		}
	}()
}

// Core application
func ReplIt(opts docopt.Opts) int {
	dir, _ := opts.String("--directory")
	if len(dir) == 0 {
		dir, _ = os.Getwd()
	}

	done := make(chan bool, 1)

	tui := NewUI()
	tui.SetTheme()

	dpath, err := filepath.Abs(dir)
	if err != nil {
		log.Fatal("replit: failed to resolve directory path")
		return 1
	}

	lang, err := opts.String("<lang>")
	if err != nil {
		log.Fatal("replit: could not read language")
		return 1
	}

	// check the editor is present; ignore the value for the moment
	_, err = GetEditor()

	if err != nil {
		panic(err)
	}

	langErr := ValidateLanguage(lang)
	if langErr != nil {
		fmt.Errorf("replit: %v", langErr)
		return 1
	}

	file, _ := opts.String("<file>")
	targetFile, err := TargetFile(file, lang)

	if err != nil {
		panic(err)
	}

	editorChan := make(chan *exec.Cmd)

	// launch an editor asyncronously
	go LaunchEditor(targetFile, editorChan)

	isTmpFile := len(file) == 0

	entrChan := make(chan *exec.Cmd)
	go StartEntr(file, lang, targetFile, dpath, entrChan)

	// terminate when a signal is received; wrap up tidily
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func(done chan bool) {
		<-sigs
		done <- true
	}(done)

	<-done

	StopReplit(entrChan, editorChan, isTmpFile, targetFile)

	return 0
}
