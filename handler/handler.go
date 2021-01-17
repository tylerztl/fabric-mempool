package handler

import (
	"context"
	"errors"
	"io/ioutil"
	"os"

	"github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/types"
	"github.com/tylerztl/fabric-mempool/mempool"
	pb "github.com/tylerztl/fabric-mempool/protos"
)

type Handler struct {
	fetcher *TxsFetcher
	mempool.Mempool
}

func (h *Handler) SubmitTransaction(ctx context.Context, etx *pb.EndorsedTransaction) (*pb.SubmitTxResponse, error) {
	err := h.Mempool.CheckTx(etx.Tx, nil, mempool.TxInfo{})
	if err != nil {
		return nil, err
	}
	return &pb.SubmitTxResponse{Status: pb.StatusCode_SUCCESS}, nil
}

func (h *Handler) FetchTransactions(ctx context.Context, ftx *pb.FetchTxsRequest) (*pb.FetchTxsResponse, error) {
	if h.Mempool.Size() <= 0 {
		return &pb.FetchTxsResponse{TxNum: 0, IsEmpty: true}, nil
	}

	expectedTxs := int(ftx.TxNum) -1

	txs := h.Mempool.ReapMaxTxs(expectedTxs)
	actualTxs := len(txs)
	isEmpty := actualTxs < expectedTxs

	logger.Info("orderer fetch transactions", "orderer", ftx.Sender,
		"actualTxs", actualTxs, "expectedTxs", expectedTxs, "mempool", h.Mempool.Size())

	orderer := h.fetcher.GetOrderer(ftx.Sender)
	if orderer == nil {
		return nil, errors.New("not found orderer connected client")
	}

	go func() {
		committedTxs := make(types.Txs, 0)
		for _, tx := range txs {
			err := orderer.broadcast(tx)
			if err != nil {
				logger.Error("failed to broadcast endorsed tx to orderer service", "error", err)
				if err = orderer.resetConnect(); err == nil && orderer.broadcast(tx) != nil {
					logger.Error("retry broadcast endorsed tx to orderer service", "orderer", ftx.Sender)
					continue
				}
			}

			committedTxs = append(committedTxs, tx)
		}
		if err := h.Mempool.Update(1, committedTxs, nil, nil, nil); err != nil {
			logger.Error("txs committed update failed", "error", err)
		}
	}()

	return &pb.FetchTxsResponse{TxNum: int32(actualTxs), IsEmpty: isEmpty}, nil
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
	cfg.Size = 10000

	pool := mempool.NewCListMempool(cfg, 0)
	pool.SetLogger(logger)

	return &Handler{
		fetcher: NewTxsFetcher(),
		Mempool: pool,
	}
}
