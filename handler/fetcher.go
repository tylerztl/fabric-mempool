package handler

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/gogo/protobuf/proto"
	cb "github.com/hyperledger/fabric/protos/common"
	ab "github.com/hyperledger/fabric/protos/orderer"
	"github.com/pkg/errors"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tylerztl/fabric-mempool/conf"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	MaxGrpcMsgSize = 1000 * 1024 * 1024
	ConnTimeout    = 30 * time.Second
	AppConf        = conf.GetAppConf().Conf
	logger         = log.NewTMLogger(log.NewSyncWriter(os.Stdout)).With("module", "fetcher")
)

type TxsFetcher struct {
	clients map[string][]*BroadcastClient // channels~orderers
}

func NewTxsFetcher() *TxsFetcher {
	runtime.GOMAXPROCS(AppConf.CPUs)

	if len(AppConf.ConnOrderers) == 0 {
		panic(" Cannot find connect orderers config")
	}
	fpath := conf.GetCryptoConfigPath(fmt.Sprintf("ordererOrganizations/example.com/orderers/%s"+"*", AppConf.ConnOrderers[0].Host))
	matches, err := filepath.Glob(fpath)
	if err != nil {
		panic(fmt.Sprintf("Cannot find filepath %s ; err: %v\n", fpath, err))
	} else if len(matches) != 1 {
		panic(fmt.Sprintf("No msp directory filepath name matches: %s\n", fpath))
	}

	engine := &TxsFetcher{
		getOrderers(),
	}

	if len(AppConf.Channels) == 0 {
		panic(" Cannot find any channel, please create channel firstly!")
	}

	return engine
}

func getOrderers() (channelClients map[string][]*BroadcastClient) {
	channelClients = make(map[string][]*BroadcastClient)
	for _, channel := range AppConf.Channels {
		ordererClients := make([]*BroadcastClient, len(AppConf.ConnOrderers))
		for i, orderer := range AppConf.ConnOrderers {
			var serverAddr string
			if AppConf.Local {
				serverAddr = fmt.Sprintf("localhost:%d", orderer.Port)
			} else {
				serverAddr = fmt.Sprintf("%s:%d", orderer.Host, orderer.Port)
			}

			fpath := conf.GetCryptoConfigPath(fmt.Sprintf("ordererOrganizations/example.com/orderers/%s"+"*", orderer.Host))
			matches, err := filepath.Glob(fpath)
			if err != nil {
				panic(fmt.Sprintf("Cannot find filepath %s ; err: %v\n", fpath, err))
			} else if len(matches) != 1 {
				panic(fmt.Sprintf("No msp directory filepath name matches: %s\n", fpath))
			}

			var dialOpts []grpc.DialOption
			dialOpts = append(dialOpts, grpc.WithDefaultCallOptions(grpc.MaxCallSendMsgSize(MaxGrpcMsgSize),
				grpc.MaxCallRecvMsgSize(MaxGrpcMsgSize)))
			if AppConf.TlsEnabled {
				creds, err := credentials.NewClientTLSFromFile(fmt.Sprintf("%s/tls/ca.crt", matches[0]), orderer.Host)
				if err != nil {
					panic(fmt.Sprintf("Error creating grpc tls client creds, serverAddr %s, err: %v", serverAddr, err))
				}
				dialOpts = append(dialOpts, grpc.WithTransportCredentials(creds))
			} else {
				dialOpts = append(dialOpts, grpc.WithInsecure())
			}

			ctx, _ := context.WithTimeout(context.Background(), ConnTimeout)
			//defer cancel()

			ordererConn, err := grpc.DialContext(ctx, serverAddr, dialOpts...)
			if err != nil {
				panic(fmt.Sprintf("Error connecting (grpc) to %s, err: %v", serverAddr, err))
			}

			client, err := ab.NewAtomicBroadcastClient(ordererConn).Broadcast(context.TODO())
			if err != nil {
				panic(fmt.Sprintf("Error creating broadcast client for orderer[%s] , err: %v", serverAddr, err))
			}

			logger.Info("Connected orderer service", "ordererAddr", serverAddr, "channel", channel)

			ordererClients[i] = &BroadcastClient{
				client: client,
			}
		}
		channelClients[channel] = ordererClients
	}

	return
}

type BroadcastClient struct {
	client ab.AtomicBroadcast_BroadcastClient
	mutex  sync.Mutex
}

func (b *BroadcastClient) broadcast(transaction []byte) error {
	env := &cb.Envelope{}
	err := proto.Unmarshal(transaction, env)
	if err != nil {
		return err
	}

	b.mutex.Lock()
	defer b.mutex.Unlock()

	done := make(chan error)
	go func() {
		done <- b.getAck()
	}()
	if err := b.client.Send(env); err != nil {
		return errors.WithMessage(err, "could not send")
	}

	return <-done
}

func (b *BroadcastClient) getAck() error {
	msg, err := b.client.Recv()
	if err != nil {
		return err
	}
	if msg.Status != cb.Status_SUCCESS {
		return fmt.Errorf("catch unexpected status: %v", msg.Status)
	}
	return nil
}
