package main

import (
	"github.com/tylerztl/fabric-mempool/conf"
	"net"
	"os"

	pb "github.com/hyperledger/fabric/protos/common"
	"github.com/spf13/cobra"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tylerztl/fabric-mempool/handler"
	"google.golang.org/grpc"
)

var logger = log.NewTMLogger(log.NewSyncWriter(os.Stdout))

var rootCmd = &cobra.Command{
	Use:   "grpc",
	Short: "Run the fabric-mempool component server",
}

var distributeConfig = conf.DistributeConfig{}

var serverCmd = &cobra.Command{
	Use:   "start",
	Short: "Run the gRPC fabric-sdk server",
	Run: func(cmd *cobra.Command, args []string) {
		err := Run()
		if err != nil {
			panic(err)
		}
	},
}

func init() {
	serverCmd.Flags().StringVarP(&ServerPort, "port", "p", "8080", "server port")
	serverCmd.Flags().IntVarP(&distributeConfig.DistributionType, "distribute", "d", 0, "distribution type")
	rootCmd.AddCommand(serverCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(-1)
	}
}

var (
	ServerPort string
	EndPoint   string
)

func Run() error {
	EndPoint = ":" + ServerPort
	conn, err := net.Listen("tcp", EndPoint)
	if err != nil {
		logger.Error("TCP Listen err:%s", err)
	}

	srv := newGrpc()
	logger.Info("Fabric mempool service running", "listenPort", ServerPort)

	if err = srv.Serve(conn); err != nil {
		logger.Error("ListenAndServe: %s", err)
	}
	return err
}

func newGrpc() *grpc.Server {
	server := grpc.NewServer()
	// TODO
	pb.RegisterMempoolServer(server, handler.NewHandler(&distributeConfig))

	return server
}
