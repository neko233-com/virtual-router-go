package main

import (
	"log"

	"virtual-router-go/VirtualRouterClient"
)

func main() {
	client, err := VirtualRouterClient.NewClient("")
	if err != nil {
		log.Fatal(err)
	}
	if err := client.Start(); err != nil {
		log.Fatal(err)
	}

	if err := client.AwaitRpcRouterInfoFirstReady(); err != nil {
		log.Fatal(err)
	}

	client.AwaitSystemClose()
}
