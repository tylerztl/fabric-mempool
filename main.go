package main

import (
	"fmt"
	"net"
	"os"

	"github.com/spf13/cobra"
	"github.com/tylerztl/fabric-mempool/handler"
	"github.com/tylerztl/fabric-mempool/helpers"
	pb "github.com/tylerztl/fabric-mempool/protos"
	"google.golang.org/grpc"
)

var rootCmd = &cobra.Command{
	Use:   "grpc",
	Short: "Run the fabric-mempool component server",
}

var serverCmd = &cobra.Command{
	Use:   "start",
	Short: "Run the gRPC fabric-sdk server",
	Run: func(cmd *cobra.Command, args []string) {
		defer func() {
			if err := recover(); err != nil {
				fmt.Printf("Recover error : %v", err)
			}
		}()

		err := Run()
		fmt.Printf("server run error : %v", err)
	},
}

func init() {
	serverCmd.Flags().StringVarP(&ServerPort, "port", "p", "8080", "server port")

	rootCmd.AddCommand(serverCmd)
}

func main() {
	handler.NewHandler()

	if err := rootCmd.Execute(); err != nil {
		os.Exit(-1)
	}
}

var (
	ServerPort string
	EndPoint   string
)

var logger = helpers.GetLogger()

func Run() (err error) {
	EndPoint = ":" + ServerPort
	conn, err := net.Listen("tcp", EndPoint)
	if err != nil {
		logger.Error("TCP Listen err:%s", err)
	}

	srv := newGrpc()
	logger.Info("gRPC and https listen on: %s", ServerPort)

	if err = srv.Serve(conn); err != nil {
		logger.Error("ListenAndServe: %s", err)
	}

	return err
}

func newGrpc() *grpc.Server {
	server := grpc.NewServer()
	// TODO
	pb.RegisterMempoolServer(server, handler.NewHandler())

	return server
}
