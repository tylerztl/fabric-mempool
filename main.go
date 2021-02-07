package main

import (
	"github.com/gin-gonic/gin"
	"github.com/tylerztl/fabric-mempool/conf"
	"net"
	"os"
	"sync"

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
	serverCmd.Flags().StringVarP(&RestPort, "rest", "r", ":80", "rest server port")
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
	RestPort   string
)

func Run() error {
	EndPoint = ":" + ServerPort
	conn, err := net.Listen("tcp", EndPoint)
	if err != nil {
		logger.Error("TCP Listen err:%s", err)
	}

	rpcHandler := handler.NewHandler(&distributeConfig)
	restHandler := handler.NewRestHandler(&distributeConfig, rpcHandler)
	r := gin.Default()
	restHandler.Register(r)
	srv := newGrpc(rpcHandler)
	logger.Info("Fabric mempool service running", "listenPort", ServerPort)

	wg := new(sync.WaitGroup)
	wg.Add(2)
	go func() {
		defer wg.Done()
		if err = srv.Serve(conn); err != nil {
			logger.Error("ListenAndServe: %s", err)
		}
	}()
	go func() {
		defer wg.Done()
		if err = r.Run(RestPort); err != nil {
			logger.Error("Rest Server: %s", err)
		}
	}()
	wg.Wait()
	return err
}

func newGrpc(rpcHandler *handler.Handler) *grpc.Server {
	server := grpc.NewServer()
	// TODO
	pb.RegisterMempoolServer(server, rpcHandler)

	return server
}
