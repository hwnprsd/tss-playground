package main

import (
	"time"

	"github.com/hwnprsd/tss/node"
)

// TODO:
// 1. Start a GRPC server
// 2. Define protobuf messages / services
// 3. Spin up random clients & communicate

func main() {
	go makeAndConnectNode("id1", ":3456", []string{})
	time.Sleep(time.Second)
	go makeAndConnectNode("id2", ":3457", []string{":3456"})
	time.Sleep(time.Second)
	go makeAndConnectNode("id3", ":3458", []string{":3457", ":3456"})
	time.Sleep(time.Second)
	go makeAndConnectNode("id4", ":3459", []string{":3457", ":3458"})
	time.Sleep(time.Second)
	go makeAndConnectNode("id5", ":3460", []string{":3456"})
	select {}
}

func makeAndConnectNode(id string, listenAddr string, knownAddresses []string) *node.Node {
	n := node.NewNode(id)
	go n.Start(listenAddr, knownAddresses)
	go func() {
		for {
			time.Sleep(time.Second * 5)
			n.LogPeers()
		}
	}()
	return n
}
