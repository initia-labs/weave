#! /bin/bash

sudo apt install make
sudo apt-get install lz4
sudo snap install go --classic
export GOPATH=$HOME/go
export PATH=$PATH:$GOROOT/bin:$GOPATH/bin

git clone https://github.com/initia-labs/weave.git
cd weave
make install
