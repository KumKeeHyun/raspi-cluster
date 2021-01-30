# Install

#### 1. install docker and kubernetes
before running the script, please check and modify specific version
- default: `DOCKER_VER=5:19.03.12~3-0~ubuntu-focal`, `K8S_VER=1.20.0-00`
```
$ sh setup.sh
```

if you want to check version
- list docker version
```
$ sudo apt-cache madison docker-ce
docker-ce | 5:20.10.1~3-0~ubuntu-focal | https://download.docker.com/linux/ubuntu focal/stable arm64 Packages
docker-ce | 5:20.10.0~3-0~ubuntu-focal | https://download.docker.com/linux/ubuntu focal/stable arm64 Packages
docker-ce | 5:19.03.14~3-0~ubuntu-focal | https://download.docker.com/linux/ubuntu focal/stable arm64 Packages
docker-ce | 5:19.03.13~3-0~ubuntu-focal | https://download.docker.com/linux/ubuntu focal/stable arm64 Packages
```
- list kubernetes version
```
$ sudo apt list -a kubeadm
Listing... Done
kubeadm/kubernetes-xenial,now 1.20.0-00 arm64 [installed]
kubeadm/kubernetes-xenial 1.19.5-00 arm64
kubeadm/kubernetes-xenial 1.19.4-00 arm64
kubeadm/kubernetes-xenial 1.19.3-00 arm64
```

#### 2. Enable Cgroups
edit `/boot/firmware/cmdline.txt` and add options
```
cgroup_enable=memory swapaccount=1 cgroup_memory=1 cgroup_enable=cpuset
```
and reboot
```
$ sudo reboot
```

#### 3. disable swap
```
$ sudo swapoff -a
```