package handler

import (
	"context"
	"errors"
	"fmt"
	"github.com/tylerztl/fabric-mempool/conf"
	"github.com/tylerztl/fabric-mempool/protoutil"
	"io/ioutil"
	"math/big"
	"os"

	pb "github.com/hyperledger/fabric/protos/common"
	//"github.com/tendermint/tendermint/types"
	"github.com/tendermint/tendermint/config"
	"github.com/tylerztl/fabric-mempool/mempool"
)

type Handler struct {
	fetcher          *TxsFetcher
	distributeConfig *conf.DistributeConfig
	mempool.Mempool
}

func (h *Handler) SubmitTransaction(ctx context.Context, etx *pb.EndorsedTransaction) (*pb.SubmitTxResponse, error) {
	err := h.Mempool.CheckTx(etx.Tx, nil, mempool.TxInfo{})
	if err != nil {
		return nil, err
	}
	return &pb.SubmitTxResponse{Status: pb.StatusCode_SUCCESS}, nil
}

// distribute check tax add to one orderer or average all orderer
func (h *Handler) distribute(tax *big.Int, orderer *BroadcastClient) {
	if h.distributeConfig.DistributionType == 1 {
		orderers := h.fetcher.GetOrderers()
		ordererCount := big.NewInt(int64(len(orderers)))
		// if less some tax after average tax, add to orderer which deal the order
		average := new(big.Int).Div(tax, ordererCount)
		less := new(big.Int).Sub(tax, new(big.Int).Mul(ordererCount, average))
		orderer.AddTax(less)
		for _, item := range orderers {
			item.AddTax(average)
		}
	} else {
		orderer.AddTax(tax)
	}
}

func (h *Handler) GetOrdererLog(name string) (string, error) {
	orderer := h.fetcher.GetOrderer(name)
	if orderer == nil {
		return "", errors.New("not found orderer connected client")
	}
	return orderer.LogOut(), nil
}

func (h *Handler) FetchTransactions(ctx context.Context, ftx *pb.FetchTxsRequest) (*pb.FetchTxsResponse, error) {
	if h.Mempool.Size() <= 0 {
		return &pb.FetchTxsResponse{TxNum: 0, IsEmpty: true}, nil
	}

	expectedTxs := int(ftx.TxNum)

	txs := h.Mempool.ReapMaxTxs(expectedTxs - 1)
	actualTxs := len(txs)
	isEmpty := actualTxs < expectedTxs

	logger.Info("orderer fetch transactions", "orderer", ftx.Sender,
		"actualTxs", actualTxs, "expectedTxs", expectedTxs, "mempool", h.Mempool.Size())

	orderer := h.fetcher.GetOrderer(ftx.Sender)
	if orderer == nil {
		return nil, errors.New("not found orderer connected client")
	}

	for _, tx := range txs {
		fee, err := protoutil.GetTxFeeFromEnvelope(tx)
		if err != nil {
			fmt.Printf("Unmarshal tx failed: %s", err)
			continue
		}
		h.distribute(fee, orderer)
		logger.Info("Unmarshal tx", "fee", fee)
	}

	// TODO
	//go func() {
	//	committedTxs := make(types.Txs, 0)
	//	for _, tx := range txs {
	//		err := orderer.broadcast(tx)
	//		if err != nil {
	//			logger.Error("failed to broadcast endorsed tx to orderer service", "error", err)
	//			if err = orderer.resetConnect(); err == nil && orderer.broadcast(tx) != nil {
	//				logger.Error("retry broadcast endorsed tx to orderer service", "orderer", ftx.Sender)
	//				continue
	//			}
	//		}
	//
	//		committedTxs = append(committedTxs, tx)
	//	}
	//	if err := h.Mempool.Update(1, committedTxs, nil, nil, nil); err != nil {
	//		logger.Error("txs committed update failed", "error", err)
	//	}
	//}()

	if err := h.Mempool.Update(1, txs, nil, nil, nil); err != nil {
		logger.Error("txs committed update failed", "error", err)
	}

	return &pb.FetchTxsResponse{TxNum: int32(actualTxs), IsEmpty: isEmpty}, nil
}

func NewHandler(distributeConfig *conf.DistributeConfig) *Handler {
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
		fetcher:          NewTxsFetcher(distributeConfig),
		Mempool:          pool,
		distributeConfig: distributeConfig,
	}
}
