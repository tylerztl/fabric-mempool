package handler

import (
	"context"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/common"
	pbpeer "github.com/hyperledger/fabric/protos/peer"
	putils "github.com/hyperledger/fabric/protos/utils"
	"github.com/pkg/errors"
	"github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/types"
	"github.com/tylerztl/fabric-mempool/conf"
	"github.com/tylerztl/fabric-mempool/mempool"
	"github.com/tylerztl/fabric-mempool/protoutil"
)

type Handler struct {
	fetcher          *TxsFetcher
	distributeConfig *conf.DistributeConfig
	sortConfig       *conf.SortConfig
	mempool.Mempool
	endorser pbpeer.EndorserClient
	signer   *Crypto
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

func (h *Handler) GetOrdererInfoList() conf.OrdererFeedback {
	list := make([]*conf.Feedback, 0)
	for k, v := range h.fetcher.clients {
		list = append(list, &conf.Feedback{
			Orderer:   k,
			Capacity:  v.capacity,
			FeeReward: v.totalTax.String(),
		})
	}
	return conf.OrdererFeedback{
		Lists: list,
	}
}

// ChangeDistribute change distribution type
func (h *Handler) ChangeDistribute(config *conf.DistributeConfig) {
	h.distributeConfig.DistributionType = config.DistributionType
	logger.Info("change transaction allocation rule", "allocation-rule", config.String())
}

func (h *Handler) ChangeSortSwitch(config *conf.SortConfig) {
	h.sortConfig.SortSwitch = config.SortSwitch
	logger.Info("change transaction sorting switch", " sorting-rule", config.String())
}

func (h *Handler) ChangeOrdererCapacity(config *conf.OrdererCapacityConfig) error {
	orderer := h.fetcher.GetOrderer(config.Orderer)
	if orderer == nil {
		logger.Error("Not found orderer", "name", config.Orderer)
		return errors.New("not found orderer")
	}

	orderer.capacity = config.Capacity
	logger.Info("change orderer capacity", "ordererName", config.Orderer, "new capacity", config.Capacity)
	return nil
}

func (h *Handler) Invoke(channelID, ccID, feeLimit string, args ...string) error {
	var argsInByte [][]byte
	for _, arg := range args {
		argsInByte = append(argsInByte, []byte(arg))
	}

	creator, _ := h.signer.Serialize()
	spec := &pbpeer.ChaincodeSpec{
		Type:        pbpeer.ChaincodeSpec_GOLANG,
		ChaincodeId: &pbpeer.ChaincodeID{Name: ccID},
		Input: &pbpeer.ChaincodeInput{
			Args: argsInByte,
		},
	}
	invocation := &pbpeer.ChaincodeInvocationSpec{ChaincodeSpec: spec}
	prop, _, err := putils.CreateChaincodeProposalWithTxIDAndTransient(pb.HeaderType_ENDORSER_TRANSACTION, channelID, feeLimit, invocation, creator, "", nil)
	if err != nil {
		return errors.WithMessage(err, "error creating proposal")
	}

	signedProp, err := GetSignedProposal(prop, h.signer)
	if err != nil {
		return errors.WithMessage(err, "error creating signed proposal")
	}

	proposalResp, err := h.endorser.ProcessProposal(context.Background(), signedProp)
	if err != nil {
		return err
	}

	if proposalResp.Response.Status >= shim.ERRORTHRESHOLD {
		return errors.WithMessage(err, fmt.Sprintf("error proposal responses received, %s", proposalResp.Response.Message))
	}

	// assemble a signed transaction (it's an Envelope message)
	env, err := CreateSignedTx(prop, h.signer, proposalResp)
	if err != nil {
		return errors.WithMessage(err, "could not assemble transaction")
	}
	envBytes, err := proto.Marshal(env)
	if err != nil {
		return err
	}
	if err := h.Mempool.CheckTx(envBytes, nil, mempool.TxInfo{}); err != nil {
		logger.Error("failed to add transaction", "error", err)
	}

	return nil
}

func (h *Handler) FetchTransactions(ctx context.Context, ftx *pb.FetchTxsRequest) (*pb.FetchTxsResponse, error) {
	if h.Mempool.Size() <= 0 {
		return &pb.FetchTxsResponse{TxNum: 0, IsEmpty: true}, nil
	}

	orderer := h.fetcher.GetOrderer(ftx.Requester)
	if orderer == nil {
		return nil, errors.New("not found orderer connected client")
	}
	expectedTxs := orderer.capacity

	var txs types.Txs
	if h.sortConfig.SortSwitch {
		txs = h.Mempool.ReapMaxTxsBySort(expectedTxs)
	} else {
		txs = h.Mempool.ReapMaxTxs(expectedTxs - 1)
	}
	actualTxs := len(txs)
	isEmpty := actualTxs < expectedTxs

	logger.Info("Fetched unconfirmed transactions for orderer", "OrdererName", ftx.Requester,
		"actualTxs", actualTxs, "capacity", expectedTxs, "mempool", h.Mempool.Size(), "blockHeight", ftx.BlockHeight)

	for i, tx := range txs {
		fee, txId, err := protoutil.GetTxFeeFromEnvelope(tx)
		if err != nil {
			fmt.Printf("Unmarshal tx failed: %s", err)
			continue
		}
		h.distribute(fee, orderer)
		logger.Info("Fetched tx detail", "index", i, "txId", txId, "fee", fee)
	}
	orderer.log()

	go func() {
		committedTxs := make(types.Txs, 0)
		for _, tx := range txs {
			err := orderer.broadcast(tx)
			if err != nil {
				logger.Error("failed to broadcast endorsed tx to orderer service", "error", err)
				if err = orderer.resetConnect(); err == nil && orderer.broadcast(tx) != nil {
					logger.Error("retry broadcast endorsed tx to orderer service", "ordererName", ftx.Requester)
				}
			}

			committedTxs = append(committedTxs, tx)
		}
		if err := h.Mempool.Update(int64(ftx.BlockHeight), committedTxs, nil, nil, nil); err != nil {
			logger.Error("txs committed update failed", "error", err)
		}
	}()

	//if err := h.Mempool.Update(1, txs, nil, nil, nil); err != nil {
	//	logger.Error("txs committed update failed", "error", err)
	//}

	return &pb.FetchTxsResponse{TxNum: int32(actualTxs), IsEmpty: isEmpty}, nil
}

func NewHandler(distributeConfig *conf.DistributeConfig, sortConfig *conf.SortConfig) *Handler {
	endorser, err := CreateEndorserClient(AppConf.Peer)
	if err != nil {
		panic(err)
	}
	signer, err := LoadCrypto(AppConf.User)
	if err != nil {
		panic(err)
	}

	// create a unique, concurrency-safe test directory under os.TempDir()
	rootDir := os.Getenv("MEMPOOL_DATA")
	if rootDir == "" {
		rootDir, err = ioutil.TempDir("", "fabric-mempool_")
		if err != nil {
			panic(err)
		}
		defer func() {
			_ = os.RemoveAll(rootDir)
		}()
	}

	cfg := config.DefaultMempoolConfig()
	cfg.CacheSize = 1000
	cfg.RootDir = rootDir
	cfg.Size = 10000000

	pool := mempool.NewCListMempool(cfg, 0)
	pool.SetLogger(logger)

	return &Handler{
		fetcher:          NewTxsFetcher(distributeConfig),
		Mempool:          pool,
		distributeConfig: distributeConfig,
		sortConfig:       sortConfig,
		endorser:         endorser,
		signer:           signer,
	}
}
