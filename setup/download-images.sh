#!/bin/bash

export BASEIMAGE_RELEASE=0.4.22

#Set MARCH variable i.e x86_64
MARCH=`uname -m`

DockerPull() {
  for IMAGES in peer orderer tools ccenv; do
      echo "==> IMAGE: $IMAGES"
      echo
      docker pull tailin/fabric-$IMAGES:1.4.10
      docker tag tailin/fabric-$IMAGES:1.4.10 hyperledger/fabric-$IMAGES
  done
}

BaseImagesPull() {
      docker pull hyperledger/fabric-baseimage:$MARCH-$BASEIMAGE_RELEASE
      docker pull hyperledger/fabric-baseos:$MARCH-$BASEIMAGE_RELEASE
      docker pull hyperledger/fabric-kafka:latest
      docker pull hyperledger/fabric-zookeeper:latest
}

echo "===> Pulling Base Images"
BaseImagesPull

echo "===> Pulling Images"
DockerPull

echo
echo "===> List out docker images"
docker images | grep hyperledger*
