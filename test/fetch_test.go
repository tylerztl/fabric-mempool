/*
Copyright Zhigui.com. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package test

import (
	"context"
	"testing"

	pb "github.com/hyperledger/fabric/protos/common"
)

func TestFetchTxs(t *testing.T) {
	conn := NewConn()
	defer conn.Close()

	c := pb.NewMempoolClient(conn)
	r, err := c.FetchTransactions(context.Background(), &pb.FetchTxsRequest{
		Requester: "orderer",
		BlockHeight:  10,
	})
	if err != nil {
		t.Error(err)
	}
	t.Log(r.TxNum, r.IsEmpty)
}
