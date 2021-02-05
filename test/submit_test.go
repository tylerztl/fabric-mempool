package test

import (
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/timestamp"
	mockmsp "github.com/hyperledger/fabric/common/mocks/msp"
	cb "github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protos/utils"
	"golang.org/x/net/context"
)

var producersWG sync.WaitGroup

func sendTransaction(payload []byte) (cb.StatusCode, error) {
	conn := NewConn()
	defer conn.Close()

	c := cb.NewMempoolClient(conn)
	context := context.Background()
	env := createSignedTx(payload)
	body, err := proto.Marshal(env)
	if err != nil {
		return cb.StatusCode_FAILED, err
	}

	if r, err := c.SubmitTransaction(context, &cb.EndorsedTransaction{Tx: body}); err != nil {
		return cb.StatusCode_FAILED, err
	} else {
		return r.Status, nil
	}
}

func syncSendTransaction(c cb.MempoolClient, payload []byte) (cb.StatusCode, error) {
	context := context.Background()

	env := createSignedTx(payload)
	body, err := proto.Marshal(env)
	if err != nil {
		return cb.StatusCode_FAILED, err
	}

	if r, err := c.SubmitTransaction(context, &cb.EndorsedTransaction{Tx: body}); err != nil {
		producersWG.Done()
		return cb.StatusCode_FAILED, err
	} else {
		producersWG.Done()
		fmt.Println("Status: ", r.Status)
		return r.Status, nil
	}
}

func createSignedTx(payload []byte) *cb.Envelope {
	prop := &peer.Proposal{}
	signID, _ := mockmsp.NewNoopMsp().GetDefaultSigningIdentity()
	signerBytes, _ := signID.Serialize()
	ccHeaderExtensionBytes, _ := proto.Marshal(&peer.ChaincodeHeaderExtension{})
	chdrBytes, _ := proto.Marshal(&cb.ChannelHeader{
		Type:    int32(cb.HeaderType_MESSAGE),
		Version: 1,
		Timestamp: &timestamp.Timestamp{
			Seconds: time.Now().Unix(),
			Nanos:   0,
		},
		ChannelId: "mychannel",
		Epoch:     1,
		Extension: ccHeaderExtensionBytes,
	})
	shdrBytes, _ := proto.Marshal(&cb.SignatureHeader{
		Creator: signerBytes,
	})
	responses := []*peer.ProposalResponse{
		{
			Payload:     payload,
			Endorsement: &peer.Endorsement{},
			Response: &peer.Response{
				Status:  200,
				Message: "response-message",
			},
		},
	}
	headerBytes, _ := proto.Marshal(&cb.Header{
		ChannelHeader:   chdrBytes,
		SignatureHeader: shdrBytes,
	})
	prop.Header = headerBytes
	env, _ := utils.CreateSignedTx(prop, signID, responses...)
	return env
}

func TestSendTransaction(t *testing.T) {
	status, err := sendTransaction([]byte("payload"))
	if status != cb.StatusCode_SUCCESS || err != nil {
		t.Error(err)
	}
}

func TestSyncSendTransaction(t *testing.T) {
	conn := NewConn()
	defer conn.Close()
	c := cb.NewMempoolClient(conn)

	txNums := 10000
	producersWG.Add(txNums)
	for i := 0; i < txNums; i++ {
		go syncSendTransaction(c, []byte(strconv.Itoa(i)))
	}
	producersWG.Wait()
}
