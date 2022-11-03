# colabware-backend

The bakend of colabware

# Learning template about GIN

https://github.com/demo-apps/go-gin-app

# First-time setup

go get -d github.com/ethereum/go-ethereum@v1.10.17
cd $GOPATH/go/pkg/mod/github.com/ethereum/go-ethereum@v1.10.17
go install ./...
export PATH=$PATH:~/go/bin

Using Homebrew...

Run `brew install ethereum`

# Request funds on Rinkeby test network

https://faucet.rinkeby.io/
https://rinkebyfaucet.com/
https://faucets.chain.link/rinkeby

# Install solc

https://pentestwiki.org/blockchain/how-to-install-solc-in-linux/

Using Homebrew...

Run `brew install solidity`

# Compile sol file;

abigen -sol inbox.sol -pkg contracts -out inbox.go

For abigen v >1.10.20...

Generate contract ABI using `solc --abi Project.sol -o build --overwrite`
Generate contract bytecode using `solc --bin Project.sol -o bin --overwrite`
Generate contract bindings using `abigen --abi ./build/Project.abi --pkg contracts --type Project --out Project.go --bin ./bin/Project.bin`

# How to run

pkill -f colabware-backend; go build; nohup ./colabware-backend >> log 2>&1 &

# Etherscan link

https://rinkeby.etherscan.io/address/0x907c3136f9689923710d2ee1983033136af390e4

# How to run MongoDB

In the root folder of the repo (e.g. colabware-backend), do: `mkdir -p mongodb/database`
Create a file called `colabware-backend/mongodb/database/docker-compose.yml` with the following content:

```
version: "3.8"
services:
  mongodb:
    image : mongo
    container_name: mongodb
    environment:
      - PUID=1000
      - PGID=1000
    volumes:
      - /$YOUR_LOCAL_REPO_PATH/colabware-backend/mongodb/database:/data/db
    ports:
      - 27017:27017
    restart: unless-stopped
```

Run `docker-compose up -d`
