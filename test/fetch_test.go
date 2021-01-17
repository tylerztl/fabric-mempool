/*
Copyright Zhigui.com. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package test

import (
	"context"
	"testing"

	pb "github.com/tylerztl/fabric-mempool/protos"
)

func TestFetchTxs(t *testing.T) {
	conn := NewConn()
	defer conn.Close()

	c := pb.NewMempoolClient(conn)
	r, err := c.FetchTransactions(context.Background(), &pb.FetchTxsRequest{
		Sender: "orderer",
		TxNum:  10,
	})
	if err != nil {
		t.Error(err)
	}
	t.Log(r.TxNum, r.IsEmpty)
}
