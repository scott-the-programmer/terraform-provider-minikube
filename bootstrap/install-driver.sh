#!/bin/bash

set -e 

VERSION="v1.31.0"

SCRIPT_DIR="$( cd -- "$( dirname -- "${BASH_SOURCE[0]:-$0}"; )" &> /dev/null && pwd 2> /dev/null; )";

driver=$1
arch=$2

if [ -z $arch ]; then
  driver+=$arch
fi;

url="https://github.com/kubernetes/minikube/releases/download/$VERSION/docker-machine-driver-$driver"
echo "Downloading $url"

mkdir -p $HOME/.minikube/bin

chown -R $SUDO_USER $HOME/.minikube
chmod -R u+wrx $HOME/.minikube

curl $url -o "/tmp/docker-machine-driver-$driver" -L
mv "/tmp/docker-machine-driver-$driver" $HOME/.minikube/bin

chown root:wheel $HOME/.minikube/bin/"docker-machine-driver-$driver"
chmod 4755 $HOME/.minikube/bin/"docker-machine-driver-$driver"
