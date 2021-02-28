module github.com/tylerztl/fabric-mempool

go 1.14

require (
	github.com/Shopify/sarama v1.27.2 // indirect
	github.com/fortytw2/leaktest v1.3.0
	github.com/fsouza/go-dockerclient v1.7.0 // indirect
	github.com/gin-gonic/gin v1.6.3
	github.com/go-kit/kit v0.10.0
	github.com/gogo/protobuf v1.3.1
	github.com/golang/protobuf v1.4.3
	github.com/hashicorp/go-version v1.2.1 // indirect
	github.com/hyperledger/fabric v1.4.10-0.20201209223044-51a3a52260f6
	github.com/hyperledger/fabric-amcl v0.0.0-20200424173818-327c9e2cf77a // indirect
	github.com/miekg/pkcs11 v1.0.3 // indirect
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.8.0
	github.com/spf13/cobra v1.1.1
	github.com/stretchr/testify v1.6.1
	github.com/sykesm/zap-logfmt v0.0.4 // indirect
	github.com/tendermint/tendermint v0.34.1
	go.uber.org/atomic v1.7.0 // indirect
	go.uber.org/zap v1.16.0 // indirect
	golang.org/x/crypto v0.0.0-20201016220609-9e8e0b390897
	golang.org/x/net v0.0.0-20201224014010-6772e930b67b
	google.golang.org/grpc v1.34.0
	gopkg.in/yaml.v2 v2.4.0
)

replace github.com/hyperledger/fabric v1.4.10-0.20201209223044-51a3a52260f6 => github.com/tylerztl/fabric-cityu v1.4.11-0.20210207154207-b0ee9aed5b10
