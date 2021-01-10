package handler

import (
	"context"
	"io/ioutil"
	"os"

	"github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tylerztl/fabric-mempool/mempool"
	pb "github.com/tylerztl/fabric-mempool/protos"
)

type Handler struct {
	fetcher *TxsFetcher
	mempool.Mempool
}

func (h *Handler) SubmitTransaction(ctx context.Context, etx *pb.EndorsedTransaction) (*pb.SubmitResponse, error) {
	err := h.Mempool.CheckTx(etx.Tx, nil, mempool.TxInfo{})
	if err != nil {
		return nil, err
	}
	return &pb.SubmitResponse{Status: pb.StatusCode_SUCCESS}, nil
}

func NewHandler() *Handler {
	// create a unique, concurrency-safe test directory under os.TempDir()
	rootDir, err := ioutil.TempDir("", "fabric-mempool_")
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = os.RemoveAll(rootDir)
	}()

	cfg := config.DefaultMempoolConfig()
	cfg.CacheSize = 1000
	cfg.RootDir = rootDir

	pool := mempool.NewCListMempool(cfg, 0)
	pool.SetLogger(log.TestingLogger())

	return &Handler{
		fetcher: NewTxsFetcher(),
		Mempool: pool,
	}
}
