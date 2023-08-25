package main

import (
	"log"
	"time"

	"github.com/hwnprsd/tss/node"
)

// TODO:
// 1. Start a GRPC server
// 2. Define protobuf messages / services
// 3. Spin up random clients & communicate

func main() {
	go makeAndConnectNode(":3456", []string{}, true)
	go makeAndConnectNode(":3457", []string{":3456"}, false)
	go makeAndConnectNode(":3458", []string{":3456"}, false)
	select {}
}

func makeAndConnectNode(listenAddr string, knownAddresses []string, dkg bool) *node.Node {
	n := node.NewNode()
	go n.Start(listenAddr, knownAddresses)
	// go func() {
	// 	for {
	// 		time.Sleep(time.Second * 5)
	// 		// n.LogVersion()
	// 		// n.LogPeers()
	// 	}
	// }()
	if dkg {
		go func() {
			for {
				time.Sleep(time.Second * 5)
				if n.PeerCount() == 2 {
					log.Println("\n\nCalling DKG")
					n.InitKeygen()
					break
				}
			}
		}()
		go func() {
			time.Sleep(35 * time.Second)
			n.InitSigning([]byte("Hello World"))
			time.Sleep(2 * time.Second)
			n.InitSigning([]byte("Lull"))
		}()
	}
	return n
}
