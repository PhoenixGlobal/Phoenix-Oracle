package services

import (
	"PhoenixOracle/gophoenix/core/logger"
	"PhoenixOracle/gophoenix/core/store"
	"fmt"
	"golang.org/x/crypto/ssh/terminal"
	"os"
	"os/signal"
	"syscall"
)

func Authenticate(store *store.Store) {
	if store.KeyStore.HasAccounts() {
		checkPassword(store)
	} else {
		createAccount(store)
	}
}

func checkPassword(store *store.Store) {
	for {
		phrase := promptPassword("Enter Password:")
		if err := store.KeyStore.Unlock(phrase); err != nil {
			fmt.Printf(err.Error())
		} else {
			printGreeting()
			break
		}
	}
}

func createAccount(store *store.Store) {
	for {
		phrase := promptPassword("NewStore Password:")
		phraseConfirmation := promptPassword("Confirm Password: ")
		if phrase == phraseConfirmation {
			_, err := store.KeyStore.NewAccount(phrase)
			if err != nil {
				logger.Fatal(err)
			}
			printGreeting()
			break
		} else {
			fmt.Println("Passwords don't match. Please try again.")
		}
	}
}

func withTerminalResetter(f func()) {
	initialTermState, err := terminal.GetState(int(syscall.Stdin))
	if err != nil {
		logger.Fatal(err)
	}

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, os.Kill)
	go func() {
		<-c
		terminal.Restore(int(syscall.Stdin), initialTermState)
		os.Exit(1)
	}()

	f()
	signal.Stop(c)
}

func promptPassword(prompt string) string {
	var rval string
	withTerminalResetter(func() {
		fmt.Print(prompt)
		bytePwd, err := terminal.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			logger.Fatal(err)
		}
		fmt.Println()
		rval = string(bytePwd)
	})
	return rval
}

func printGreeting() {
	fmt.Println(`
     hello welcome to phoenix
`)
}
