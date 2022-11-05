package main

import (
	"fishpi-Golient/lib"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	go lib.WssLink()
	go signalExit()
	go lib.WssGetLiveness()
	lib.WssClient()
}

func signalExit() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func(c chan os.Signal) {
		<-c
		fmt.Println("\rCtrl c pressed in terminal client exit")
		os.Exit(0)
	}(c)
}
