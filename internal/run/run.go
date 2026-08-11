// Package run は外部コマンドの実行と行単位のログ配信。
package run

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

type Level int

const (
	LevelInfo Level = iota
	LevelOK
	LevelWarn
	LevelErr
	LevelSkip
)

type Event struct {
	Level Level
	Text  string
}

// Reporter は Event の受け取り先。nil でも安全に呼べる。
type Reporter func(Event)

func (r Reporter) emit(l Level, format string, a ...any) {
	if r == nil {
		return
	}
	r(Event{Level: l, Text: fmt.Sprintf(format, a...)})
}

func (r Reporter) Info(f string, a ...any) { r.emit(LevelInfo, f, a...) }
func (r Reporter) OK(f string, a ...any)   { r.emit(LevelOK, f, a...) }
func (r Reporter) Warn(f string, a ...any) { r.emit(LevelWarn, f, a...) }
func (r Reporter) Err(f string, a ...any)  { r.emit(LevelErr, f, a...) }
func (r Reporter) Skip(f string, a ...any) { r.emit(LevelSkip, f, a...) }

func Command(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}

// Look はコマンドが PATH にあるかを返す。
func Look(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func Output(name string, args ...string) (string, error) {
	out, err := Command(name, args...).Output()
	return string(out), err
}

func Stream(rep Reporter, name string, args ...string) error {
	cmd := Command(name, args...)

	pr, pw := io.Pipe()
	cmd.Stdout = pw
	cmd.Stderr = pw

	done := make(chan struct{})
	go func() {
		defer close(done)
		sc := bufio.NewScanner(pr)
		sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for sc.Scan() {
			line := strings.TrimRight(sc.Text(), "\r")
			if strings.TrimSpace(line) == "" {
				continue
			}
			rep.Info("  %s", line)
		}
		io.Copy(io.Discard, pr) //nolint:errcheck // 読み残しを捨てる
	}()

	err := cmd.Run()
	pw.Close()
	<-done
	return err
}
