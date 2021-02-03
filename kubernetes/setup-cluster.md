# Setup Kubernetes Cluster

## 마스터 노드 생성
```
$ kubeadm init

$ mkdir -p $HOME/.kube
$ sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
$ sudo chown $(id -u):$(id -g) $HOME/.kube/config

$ kubectl get pods -n kube-system
NAME                                  READY   STATUS    RESTARTS   AGE
coredns-74ff55c5b-7qckc               0/1     Pending   0          2m
coredns-74ff55c5b-hfdw8               0/1     Pending   0          2m
etcd-balns-k8s-1                      1/1     Running   0          2m
kube-apiserver-balns-k8s-1            1/1     Running   0          2m
kube-controller-manager-balns-k8s-1   1/1     Running   0          2m
kube-proxy-pv8h4                      1/1     Running   0          2m
kube-scheduler-balns-k8s-1            1/1     Running   0          2m

$ kubectl get node
NAME         STATUS     ROLES                 AGE       VERSION
balns-k8s-1  NotReady   control-plane,master  2m        v1.20.0
```

`kubeadm`으로 생성된 모든 구성 요소는 컨테이너로 실행된다. coredns의 상태가 Pending이고 node의 상태가 NotReady인것을 확인하자.

## 워커 노드 생성
```
$ kubeadm join 192.168.219.201:6443 --token prno7e.jh91y6gjovf8ws2l \
    --discovery-token-ca-cert-hash sha256:626ea7d4f93b5939aa292acb2681d058fbd848c8258d258955ab2f2864488029

$ kubectl get node
NAME          STATUS     ROLES                  AGE    VERSION
balns-k8s-1   NotReady   control-plane,master   3m     v1.20.0
balns-k8s-2   NotReady   <none>                 1m     v1.20.0
```

`kubeadm init`으로 마스터 노드를 생성할 때 출력된 명령어를 워커 노드에서 실행한다. 워커 노드가 추가되었지만 여전히 노드의 상태가 NotReady이다. `kubectl describe`로 노드의 정보를 출력해보면 네트워크(CNI) 플러그인이 준비되지 않았다고 출력될 것이다.

```
$ kubectl describe node balns-k8s-2
...
KubeletNotReady    runtime network not ready: NetworkReady=false
                   reason:NetworkPluginNotReady message:docker:
                   network plugin is not ready: cni config uninitialized

```

## 파드 네트워크 에드온 추가
[pod network addons](https://kubernetes.io/docs/concepts/cluster-administration/addons/)에 쿠버네티스에 사용할 수 있는 CNI 에드온 목록이 있다. 나는 `Weave Net`을 사용했다.

```
$ kubectl apply -f "https://cloud.weave.works/k8s/net?k8s-version=$(kubectl version | base64 | tr -d '\n')"

$ kubectl get pods -n kube-system
NAME                                  READY   STATUS    RESTARTS   AGE
...
weave-net-7wzjf                       2/2     Running   1          4m
weave-net-rqhsk                       2/2     Running   1          4m
```

이렇게 하면 DaemonSet 및 몇가지 네트워크, 보안 관련 리소스가 배포된다.

## 메트릭스 서버 추가
CPU, 메모리 사용량을 확인해보려고 `kubectl top`을 사용했더니 메트릭스를 사용할 수 없다고 나온다. 찾아보니 메트릭스 서버는 자동으로 설치되는게 아니라서 직접 설치해야 한다고 한다.

[출처](https://m.blog.naver.com/isc0304/221860790762)

```
$ kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/download/v0.3.7/components.yaml
```

메트릭스 서버는 실행되지만 TLS 통신이 제대로 이뤄지지 않아서 몇가지 설정을 해줘야 한다.

```
$ kubectl edit deployments.apps -n kube-system metrics-server

...
spec:
containers:
- args:
- --cert-dir=/tmp
- --secure-port=4443
- --kubelet-insecure-tls # 추가된 옵션
- --kubelet-preferred-address-types=InternalIP # 추가된 옵션
...

deployment.apps/metrics-server edited

$ kubectl top node
NAME          CPU(cores)   CPU%   MEMORY(bytes)   MEMORY%
balns-k8s-1   460m         11%    1685Mi          45%
balns-k8s-2   194m         4%     1363Mi          17%
balns-k8s-3   196m         4%     1098Mi          14%
```


## kubectl 설정
원격으로 클러스터를 사용해보자. 

우선 호스트에 kubectl을 설치하자.

```
curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl && chmod +x kubectl && sudo mv kubectl /usr/local/bin/
```

쿠버네티스 클러스터의 kubectl config 파일을 복사해오자.

```
# cluseter-ip 에 마스터 노드의 IP 주소를 입력해주자.
# 혹시나 설정파일이 충돌할 수 있어서 config2 으로 복사한다.
$ scp root@{cluster-ip}:/etc/kubernetes/admin.conf ~ /.kube /config2

# 만약 root 계정으로 접속을 못하면 /etc/kubernetes/admin.conf 대신 ~/.kube/config 를 복사해오자.
$ scp {user}@{cluster-ip}:~/.kube/config ~ /.kube /config2

$ export KUBECONFIG=~/.kube/config2

# 잘 설정됐는지 확인
$ kubectl cluster-info
```

bash 설정을 해보자. 

```
# ~/.bashrc에 source <(kubectl completion bash) 추가
$ echo 'source <(kubectl completion bash)' >>~/.bashrc

# kubectl 자동 완성 스크립트를 bash_completion.d 에 추가
$ kubectl completion bash |sudo tee /etc/bash_completion.d/kubectl

# kubectl 별칭 사용
$ echo 'alias k=kubectl' >>~/.bashrc
$ echo 'complete -F __start_kubectl k' >>~/.bashrc
```

다하고 난뒤 ~/.bashrc 상태

```
export KUBECONFIG=~/.kube/config2

source <(kubectl completion bash)

alias k=kubectl
complete -F __start_kubectl k
```
