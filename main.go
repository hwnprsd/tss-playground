package main

import (
	"fmt"
	"log"
	"time"

	"github.com/hwnprsd/tss/node"
	"github.com/spf13/cobra"
)

// TODO:
// 1. Start a GRPC server
// 2. Define protobuf messages / services
// 3. Spin up random clients & communicate

func main() {
	var port int
	var peerUrls []string
	var grpcEndpoint bool

	rootCmd := &cobra.Command{Use: "SolaceTSS", Short: "Secure MPC netowork for Account Abstraction Wallets"}

	runNodeCmd := &cobra.Command{
		Use:   "node",
		Short: "Join the Solace MPC network",
		Run: func(cmd *cobra.Command, args []string) {
			go makeAndConnectNode(fmt.Sprintf(":%d", port), peerUrls, grpcEndpoint)
			select {}
		},
	}
	runNodeCmd.Flags().StringSliceVarP(&peerUrls, "peers", "x", []string{}, "List of known peer URLs")
	runNodeCmd.Flags().IntVarP(&port, "port", "p", 8080, "Port to run the node on")
	runNodeCmd.Flags().BoolVarP(&grpcEndpoint, "public-node", "n", false, "Should run a public gRPC endpoint")

	rootCmd.AddCommand(runNodeCmd)
	rootCmd.Execute()

	// go makeAndConnectNode(":3456", []string{}, true)
	// go makeAndConnectNode(":3457", []string{":3456"}, false)
	// go makeAndConnectNode(":3458", []string{":3456"}, false)
	// select {}
}

func makeAndConnectNode(listenAddr string, knownAddresses []string, dkg bool) *node.Node {
	n := node.NewNode()
	go n.Start(listenAddr, knownAddresses, dkg)
	// go func() {
	// 	for {
	// 		time.Sleep(time.Second * 5)
	// 		// n.LogVersion()
	// 		// n.LogPeers()
	// 	}
	// }()
	// if dkg {
	if false {
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
			n.InitSigning([]byte("Hello Lulli"))
		}()
	}
	return n
}
