package main

import (
	"net"
	"net/http"
	"os"
	"sync"

	"github.com/gin-gonic/gin"
	pb "github.com/hyperledger/fabric/protos/common"
	"github.com/spf13/cobra"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tylerztl/fabric-mempool/conf"
	"github.com/tylerztl/fabric-mempool/handler"
	"google.golang.org/grpc"
)

var logger = log.NewTMLogger(log.NewSyncWriter(os.Stdout))

var rootCmd = &cobra.Command{
	Use:   "grpc",
	Short: "Run the fabric-mempool component server",
}

var distributeConfig = conf.DistributeConfig{}
var sortConfig = conf.SortConfig{}

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
	serverCmd.Flags().BoolVarP(&sortConfig.SortSwitch, "sort", "s", true, "mempool sort switch")

	importCmd.Flags().StringVarP(&FilePath, "filepath", "f", "", "数据文件所在路径")
	importCmd.Flags().IntVarP(&BatchNum, "batch", "b", 100, "每次上传的数据量（条/次）")
	importCmd.Flags().Int64VarP(&Interval, "interval", "i", 0, "请求时间间隔（纳秒）")
	rootCmd.AddCommand(serverCmd)
	rootCmd.AddCommand(importCmd)
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

	FilePath string
	BatchNum int
	Interval int64
)

func Cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		// 可将将* 替换为指定的域名
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, UPDATE")
		c.Header("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization")
		c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Cache-Control, Content-Language, Content-Type")
		c.Header("Access-Control-Allow-Credentials", "true")
		if method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
		}
		c.Next()
	}
}

func Run() error {
	EndPoint = ":" + ServerPort
	conn, err := net.Listen("tcp", EndPoint)
	if err != nil {
		logger.Error("TCP Listen err:%s", err)
		return err
	}

	rpcHandler := handler.NewHandler(&distributeConfig, &sortConfig)
	restHandler := handler.NewRestHandler(&distributeConfig, rpcHandler)
	r := gin.Default()
	r.Use(Cors())
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
