# colabware-backend
The bakend of colabware

# Learning template about GIN
https://github.com/demo-apps/go-gin-app

# First-time setup
go get -d github.com/ethereum/go-ethereum@v1.10.17
cd $GOPATH/go/pkg/mod/github.com/ethereum/go-ethereum@v1.10.17
go install ./...
export PATH=$PATH:~/go/bin

# Request funds on Rinkeby test network
https://faucet.rinkeby.io/
https://rinkebyfaucet.com/
https://faucets.chain.link/rinkeby

# Install solc
https://pentestwiki.org/blockchain/how-to-install-solc-in-linux/

# Compile sol filesefekjl; 
abigen -sol inbox.sol -pkg contracts -out inbox.go

# How to run
go build; ./colabware-backend

# Etherscan link
https://rinkeby.etherscan.io/address/0x907c3136f9689923710d2ee1983033136af390e4