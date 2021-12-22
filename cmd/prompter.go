package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"PhoenixOracle/lib/logger"

	"golang.org/x/term"
)

type Prompter interface {
	Prompt(string) string
	PasswordPrompt(string) string
	IsTerminal() bool
}

type terminalPrompter struct{}

func NewTerminalPrompter() Prompter {
	return terminalPrompter{}
}

func (tp terminalPrompter) Prompt(prompt string) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(prompt)
	line, err := reader.ReadString('\n')
	if err != nil {
		logger.Fatal(err)
	}
	clearLine()
	return strings.TrimSpace(line)
}

func (tp terminalPrompter) PasswordPrompt(prompt string) string {
	var rval string
	withTerminalResetter(func() {
		fmt.Print(prompt)
		bytePwd, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			logger.Fatal(err)
		}
		clearLine()
		rval = string(bytePwd)
	})
	return rval
}

func (tp terminalPrompter) IsTerminal() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

func withTerminalResetter(f func()) {
	osSafeStdin := int(os.Stdin.Fd())

	initialTermState, err := term.GetState(osSafeStdin)
	if err != nil {
		logger.Fatal(err)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		err := term.Restore(osSafeStdin, initialTermState)
		logger.ErrorIf(err, "failed when restore terminal")
		os.Exit(1)
	}()

	f()
	signal.Stop(c)
}

func clearLine() {
	fmt.Printf("\r" + strings.Repeat(" ", 60) + "\r")
}
