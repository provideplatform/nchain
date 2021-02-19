module github.com/provideapp/nchain

go 1.13

require (
	github.com/FactomProject/go-bip32 v0.3.5
	github.com/StackExchange/wmi v0.0.0-20190523213315-cbe66965904d // indirect
	github.com/btcsuite/btcd v0.21.0-beta
	github.com/btcsuite/btcutil v1.0.2
	github.com/cespare/cp v1.1.1 // indirect
	github.com/deckarep/golang-set v1.7.2-0.20180927150649-699df6a3acf6 // indirect
	github.com/ethereum/go-ethereum v1.9.22
	github.com/fjl/memsize v0.0.0-20190710130421-bcb5799ab5e5 // indirect
	github.com/gballet/go-libpcsclite v0.0.0-20191108122812-4678299bea08 // indirect
	github.com/gin-gonic/gin v1.6.3
	github.com/go-ole/go-ole v1.2.4 // indirect
	github.com/golang-migrate/migrate v3.5.4+incompatible
	github.com/gorilla/websocket v1.4.1
	github.com/ipfs/go-cid v0.0.4 // indirect
	github.com/ipfs/go-ipfs-api v0.0.2
	github.com/ipfs/go-ipfs-files v0.0.6 // indirect
	github.com/jackpal/go-nat-pmp v1.0.2 // indirect
	github.com/jinzhu/gorm v1.9.16
	github.com/joho/godotenv v1.3.0
	github.com/karalabe/usb v0.0.0-20191104083709-911d15fe12a9 // indirect
	github.com/kthomas/go-aws-config v0.0.0-20200121043457-1931a324f423
	github.com/kthomas/go-db-config v0.0.0-20200612131637-ec0436a9685e
	github.com/kthomas/go-logger v0.0.0-20200602072946-d7d72dfc2531
	github.com/kthomas/go-natsutil v0.0.0-20200602073459-388e1f070b05
	github.com/kthomas/go-pgputil v0.0.0-20200602073402-784e96083943
	github.com/kthomas/go-redisutil v0.0.0-20200602073431-aa49de17e9ff
	github.com/kthomas/go.uuid v1.2.1-0.20190324131420-28d1fa77e9a4
	github.com/libp2p/go-libp2p-core v0.3.0 // indirect
	github.com/libp2p/go-libp2p-metrics v0.1.0 // indirect
	github.com/libp2p/go-libp2p-peer v0.2.0 // indirect
	github.com/mattn/go-colorable v0.1.2 // indirect
	github.com/miguelmota/go-ethereum-hdwallet v0.0.0-20200123000308-a60dcd172b4c
	github.com/minio/sha256-simd v0.1.2-0.20190917233721-f675151bb5e1 // indirect
	github.com/multiformats/go-multiaddr-net v0.1.1 // indirect
	github.com/multiformats/go-varint v0.0.2 // indirect
	github.com/nats-io/stan.go v0.7.0
	github.com/olekukonko/tablewriter v0.0.3 // indirect
	github.com/onsi/ginkgo v1.14.0
	github.com/onsi/gomega v1.10.1
	github.com/prometheus/tsdb v0.10.0 // indirect
	github.com/provideapp/ident v0.0.0-00010101000000-000000000000
	github.com/provideservices/provide-go v0.0.0-20201207152725-cbbcf3a37eb9
	github.com/spaolacci/murmur3 v1.1.1-0.20190317074736-539464a789e9 // indirect
	github.com/status-im/keycard-go v0.0.0-20191119114148-6dd40a46baa0 // indirect
	github.com/stretchr/testify v1.5.1 // indirect
	go.mongodb.org/mongo-driver v1.3.3
)

replace github.com/provideapp/ident => ../ident
