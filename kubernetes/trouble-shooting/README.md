# Trouble Shooting

## node status NotReady
`--pod-netword-cidr` 지정하지 않을 경우 node 들의 상태가 NotReady... coredns 컨테이너는 pending 상태에서 멈춰있었음.
```
$ kubeadm init --pod-network-cidr=10.224.0.0/16
```
cidr 옵션을 추가해주고 다시 실행했더니 Ready 상태로 변경됨.

- pod-network-cidr 에 대해서 알아보자
    - 네트워크 주소는 하나같이 `10.224.0.0/16`를 쓰던데 다른걸 써도 상관 없을까?

## coredns's status ContainerCreating
- https://stackoverflow.com/questions/59558611/core-dns-stuck-in-containercreating-status
구글링을 해보면 하나같이 `flannel`, `cni` 에 대한 말밖에 없다... 그들의 말대로
```
$ kubectl apply -f https://raw.githubusercontent.com/coreos/flannel/2140ac876ef134e0ed5af15c65e414cf26827915/Documentation/kube-flannel.yml
```
을 해주었더니 조금의 warning은 뜨지만 coredns가 Running 상태로 바뀌었다!

```
podsecuritypolicy.policy/psp.flannel.unprivileged created
Warning: rbac.authorization.k8s.io/v1beta1 ClusterRole is deprecated in v1.17+, unavailable in v1.22+; use rbac.authorization.k8s.io/v1 ClusterRole
clusterrole.rbac.authorization.k8s.io/flannel created
Warning: rbac.authorization.k8s.io/v1beta1 ClusterRoleBinding is deprecated in v1.17+, unavailable in v1.22+; use rbac.authorization.k8s.io/v1 ClusterRoleBinding
clusterrolebinding.rbac.authorization.k8s.io/flannel created
serviceaccount/flannel created
configmap/kube-flannel-cfg created
daemonset.apps/kube-flannel-ds-amd64 created
daemonset.apps/kube-flannel-ds-arm64 created
daemonset.apps/kube-flannel-ds-arm created
daemonset.apps/kube-flannel-ds-ppc64le created
daemonset.apps/kube-flannel-ds-s390x created
```

- flannel 에 대해서 알아보자