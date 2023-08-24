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
	go makeAndConnectNode(":3456", []string{})
	time.Sleep(time.Second)
	go makeAndConnectNode(":3457", []string{":3456"})
	time.Sleep(time.Second * 2)
	go makeAndConnectNode(":3458", []string{":3456"})
	select {}
}

func makeAndConnectNode(listenAddr string, knownAddresses []string) *node.Node {
	n := node.NewNode()
	go n.Start(listenAddr, knownAddresses)
	go func() {
		for {
			time.Sleep(time.Second * 5)
			n.LogVersion()
			n.LogPeers()
		}
	}()
	return n
}
