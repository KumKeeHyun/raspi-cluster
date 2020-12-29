#!/bin/sh

DOCKER_VER=5:19.03.12~3-0~ubuntu-focal
K8S_VER=1.20.0-00

sudo apt update
sudo apt -y upgrade

# set docker pakage path
sudo apt-get install -y apt-transport-https ca-certificates curl gnupg-agent software-properties-common
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo apt-key add -
sudo add-apt-repository "deb [arch=arm64] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable"
sudo apt-get update

# install docker specific version 
sudo apt-get install -y docker-ce=${DOCKER_VER} docker-ce-cli=${DOCKER_VER} containerd.io

# set docker cgroupdriver cgroupfs -> systemd
cat <<EOF | sudo tee /etc/docker/daemon.json
{
  "exec-opts": ["native.cgroupdriver=systemd"],
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "100m"
  },
  "storage-driver": "overlay2"
}
EOF

sudo mkdir -p /etc/systemd/system/docker.service.d
sudo systemctl daemon-reload
sudo service docker restart

# set kubernetes pakage path and some configuration
sudo apt-get update && sudo apt-get install -y apt-transport-https curl
curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key add -

cat <<EOF | sudo tee /etc/apt/sources.list.d/kubernetes.list
deb https://apt.kubernetes.io/ kubernetes-xenial main
EOF

cat <<EOF | sudo tee /etc/sysctl.d/k8s.conf
net.bridge.bridge-nf-call-ip6tables = 1
net.bridge.bridge-nf-call-iptables = 1
EOF

sudo sysctl --system
sudo apt-get update

# install kubernetes specific version 
sudo apt-get install -y kubelet=${K8S_VER} kubeadm=${K8S_VER} kubectl=${K8S_VER}
sudo apt-mark hold kubelet kubeadm kubectl
