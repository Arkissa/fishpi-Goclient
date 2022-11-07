package main

import (
	"fishpi-Golient/lib"
	"log"
)

func main() {
	fish, err := lib.NewFishpi()
	if err != nil {
		log.Println(err)
		return
	}
	go fish.WssLink()
	fish.WssClient()
}
