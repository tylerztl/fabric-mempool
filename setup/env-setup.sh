#! /bin/bash

# now config for ubuntu 16-04

#####################################
# Install and setup Docker services #
#####################################

echo "Install docker"

sudo apt-get update

sudo apt-get -y install apt-transport-https ca-certificates curl software-properties-common

curl -fsSL http://mirrors.aliyun.com/docker-ce/linux/ubuntu/gpg | sudo apt-key add -

sudo add-apt-repository "deb [arch=amd64] http://mirrors.aliyun.com/docker-ce/linux/ubuntu $(lsb_release -cs) stable"

sudo apt-get -y update

sudo apt-get -y install docker-ce

# config docker

sudo tee /etc/docker/daemon.json <<-'EOF'
{
  "registry-mirrors": ["http://2743e10c.m.daocloud.io"]
}
EOF

sudo systemctl daemon-reload

sudo systemctl restart docker

################################################
# Install  docker-compose #
################################################

echo "Install docker-compose"
sudo apt-get -y install python-pip
export LC_ALL=C
sudo pip install --upgrade pip
sudo pip install docker-compose
