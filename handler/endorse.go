/*
Copyright Zhigui.com. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package handler

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/asn1"
	"encoding/pem"
	"fmt"
	"github.com/tylerztl/fabric-mempool/conf"
	"io/ioutil"
	"math/big"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/hyperledger/fabric/bccsp/utils"
	"github.com/hyperledger/fabric/common/crypto"
	"github.com/hyperledger/fabric/core/comm"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/msp"
	"github.com/hyperledger/fabric/protos/peer"
	putils "github.com/hyperledger/fabric/protos/utils"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

type Node struct {
	Addr      string `yaml:"addr"`
	TLSCACert string `yaml:"tls_ca_cert"`
}

type ECDSASignature struct {
	R, S *big.Int
}

func CreateEndorserClient(node *conf.PeerInfo) (peer.EndorserClient, error) {
	conn, err := DailConnection(node)
	if err != nil {
		return nil, err
	}
	return peer.NewEndorserClient(conn), nil
}

func DailConnection(node *conf.PeerInfo) (*grpc.ClientConn, error) {
	gRPCClient, err := CreateGRPCClient(node)
	if err != nil {
		return nil, err
	}
	var connError error
	var conn *grpc.ClientConn
	for i := 1; i <= 3; i++ {
		conn, connError = gRPCClient.NewConnection(node.Addr, "", func(tlsConfig *tls.Config) { tlsConfig.InsecureSkipVerify = true })
		if connError == nil {
			return conn, nil
		} else {
			logger.Error(fmt.Sprintf("%d of 3 attempts to make connection to %s, details: %s", i, node.Addr, connError))
		}
	}
	return nil, errors.Wrapf(connError, "error connecting to %s", node.Addr)
}

func CreateGRPCClient(node *conf.PeerInfo) (*comm.GRPCClient, error) {
	tlsCacert, err := GetTLSCACerts(node.TLSCACert)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to load TLS CA Cert %s", node.TLSCACert)
	}

	config := comm.ClientConfig{}
	config.Timeout = 5 * time.Second
	config.SecOpts = &comm.SecureOptions{
		UseTLS:            true,
		RequireClientCert: false,
		Certificate:       tlsCacert,
	}

	grpcClient, err := comm.NewGRPCClient(config)
	//to do: unit test for this error, current fails to make case for this
	if err != nil {
		return nil, errors.Wrapf(err, "error connecting to %s", node.Addr)
	}

	return grpcClient, nil
}

type Crypto struct {
	Creator  []byte
	PrivKey  *ecdsa.PrivateKey
	SignCert *x509.Certificate
}

func (s *Crypto) Sign(message []byte) ([]byte, error) {
	ri, si, err := ecdsa.Sign(rand.Reader, s.PrivKey, digest(message))
	if err != nil {
		return nil, err
	}

	si, err = utils.ToLowS(&s.PrivKey.PublicKey, si)
	if err != nil {
		return nil, err
	}

	return asn1.Marshal(ECDSASignature{ri, si})
}

func digest(in []byte) []byte {
	h := sha256.New()
	h.Write(in)
	return h.Sum(nil)
}

func (s *Crypto) Serialize() ([]byte, error) {
	return s.Creator, nil
}

func (s *Crypto) NewSignatureHeader() (*common.SignatureHeader, error) {
	creator, err := s.Serialize()
	if err != nil {
		return nil, err
	}
	nonce, err := crypto.GetRandomNonce()
	if err != nil {
		return nil, err
	}

	return &common.SignatureHeader{
		Creator: creator,
		Nonce:   nonce,
	}, nil
}

func LoadCrypto(c *conf.UserInfo) (*Crypto, error) {
	priv, err := GetPrivateKey(c.PrivateKey)
	if err != nil {
		return nil, errors.Wrapf(err, "error loading priv key")
	}

	cert, certBytes, err := GetCertificate(c.SignCert)
	if err != nil {
		return nil, errors.Wrapf(err, "error loading certificate")
	}

	id := &msp.SerializedIdentity{
		Mspid:   c.MSPID,
		IdBytes: certBytes,
	}

	name, err := proto.Marshal(id)
	if err != nil {
		return nil, errors.Wrapf(err, "error get msp id")
	}

	return &Crypto{
		Creator:  name,
		PrivKey:  priv,
		SignCert: cert,
	}, nil
}

func GetTLSCACerts(file string) ([]byte, error) {
	if len(file) == 0 {
		return nil, nil
	}

	in, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, errors.Wrapf(err, "error loading %s", file)
	}
	return in, nil
}

func GetPrivateKey(f string) (*ecdsa.PrivateKey, error) {
	in, err := ioutil.ReadFile(f)
	if err != nil {
		return nil, err
	}

	k, err := utils.PEMtoPrivateKey(in, []byte{})
	if err != nil {
		return nil, err
	}

	key, ok := k.(*ecdsa.PrivateKey)
	if !ok {
		return nil, errors.Errorf("expecting ecdsa key")
	}

	return key, nil
}

func GetCertificate(f string) (*x509.Certificate, []byte, error) {
	in, err := ioutil.ReadFile(f)
	if err != nil {
		return nil, nil, err
	}

	block, _ := pem.Decode(in)

	c, err := x509.ParseCertificate(block.Bytes)
	return c, in, err
}

// GetSignedProposal returns a signed proposal given a Proposal message and a
// signing identity
func GetSignedProposal(prop *peer.Proposal, signer *Crypto) (*peer.SignedProposal, error) {
	// check for nil argument
	if prop == nil || signer == nil {
		return nil, errors.New("nil arguments")
	}

	propBytes, err := putils.GetBytesProposal(prop)
	if err != nil {
		return nil, err
	}

	signature, err := signer.Sign(propBytes)
	if err != nil {
		return nil, err
	}

	return &peer.SignedProposal{ProposalBytes: propBytes, Signature: signature}, nil
}

// CreateSignedTx assembles an Envelope message from proposal, endorsements,
// and a signer. This function should be called by a client when it has
// collected enough endorsements for a proposal to create a transaction and
// submit it to peers for ordering
func CreateSignedTx(proposal *peer.Proposal, signer *Crypto, resps ...*peer.ProposalResponse) (*common.Envelope, error) {
	if len(resps) == 0 {
		return nil, errors.New("at least one proposal response is required")
	}

	// the original header
	hdr, err := putils.GetHeader(proposal.Header)
	if err != nil {
		return nil, err
	}

	// the original payload
	pPayl, err := putils.GetChaincodeProposalPayload(proposal.Payload)
	if err != nil {
		return nil, err
	}

	// check that the signer is the same that is referenced in the header
	// TODO: maybe worth removing?
	signerBytes, err := signer.Serialize()
	if err != nil {
		return nil, err
	}

	shdr, err := putils.GetSignatureHeader(hdr.SignatureHeader)
	if err != nil {
		return nil, err
	}

	if bytes.Compare(signerBytes, shdr.Creator) != 0 {
		return nil, errors.New("signer must be the same as the one referenced in the header")
	}

	// get header extensions so we have the visibility field
	hdrExt, err := putils.GetChaincodeHeaderExtension(hdr)
	if err != nil {
		return nil, err
	}

	// ensure that all actions are bitwise equal and that they are successful
	var a1 []byte
	for n, r := range resps {
		if n == 0 {
			a1 = r.Payload
			if r.Response.Status < 200 || r.Response.Status >= 400 {
				return nil, errors.Errorf("proposal response was not successful, error code %d, msg %s", r.Response.Status, r.Response.Message)
			}
			continue
		}

		if bytes.Compare(a1, r.Payload) != 0 {
			return nil, errors.New("ProposalResponsePayloads do not match")
		}
	}

	// fill endorsements
	endorsements := make([]*peer.Endorsement, len(resps))
	for n, r := range resps {
		endorsements[n] = r.Endorsement
	}

	// create ChaincodeEndorsedAction
	cea := &peer.ChaincodeEndorsedAction{ProposalResponsePayload: resps[0].Payload, Endorsements: endorsements}

	// obtain the bytes of the proposal payload that will go to the transaction
	propPayloadBytes, err := putils.GetBytesProposalPayloadForTx(pPayl, hdrExt.PayloadVisibility)
	if err != nil {
		return nil, err
	}

	// serialize the chaincode action payload
	cap := &peer.ChaincodeActionPayload{ChaincodeProposalPayload: propPayloadBytes, Action: cea}
	capBytes, err := putils.GetBytesChaincodeActionPayload(cap)
	if err != nil {
		return nil, err
	}

	// create a transaction
	taa := &peer.TransactionAction{Header: hdr.SignatureHeader, Payload: capBytes}
	taas := make([]*peer.TransactionAction, 1)
	taas[0] = taa
	tx := &peer.Transaction{Actions: taas}

	// serialize the tx
	txBytes, err := putils.GetBytesTransaction(tx)
	if err != nil {
		return nil, err
	}

	// create the payload
	payl := &common.Payload{Header: hdr, Data: txBytes}
	paylBytes, err := putils.GetBytesPayload(payl)
	if err != nil {
		return nil, err
	}

	// sign the payload
	sig, err := signer.Sign(paylBytes)
	if err != nil {
		return nil, err
	}

	// here's the envelope
	return &common.Envelope{Payload: paylBytes, Signature: sig}, nil
}
