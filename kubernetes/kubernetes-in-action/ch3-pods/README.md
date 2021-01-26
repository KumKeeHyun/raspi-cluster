# Chapter 3. Pods:running containers in kubernetes

## Introducing Pods
쿠버네티스는 컨테이너를 개별적으로 배포하는 대신 파드로 한번더 감싸서 운영한다. 단일 포드에 둘 이상의 컨테이너를 포함시킬 수 있지만 단일 컨테이너만 포함하는 것이 일반적이다. 파드의 핵심은 파드 안에 포함된 컨테이너들은 항상 단일 워커 노드에서 실행되는 것이다. 아래 그림과 같이 실행되지 않는다.

![image](https://user-images.githubusercontent.com/44857109/105663606-06076800-5f16-11eb-8e53-4b195d6fc868.png)

<br>

### 왜 쿠버네티스는 컨테이너를 직접 사용하지 않고 파드라는 개념을 만들었을까? 

컨테이너는 컨테이너 당 하나의 프로세스만 실행하도록 설계되었다. 단일 컨테이너에서 관련되지 않은 여러 프로세스를 실행하는 경우 각 프로세스를 관리하기 위한 노력(로그 관리, 충돌시 재시작 등)은 모두 개발자의 책임이 되어버린다. 따라서 각 프로세스는 자체 컨테이너에서 실행되어야 한다.

서로 밀접하게 관련된 프로세스를 함께 실행하고 싶을 때 여러 프로세스를 단일 컨테이너로 그룹화하면 안되기 때문에 여러 컨테이너를 함께 관리하기 위한 상위 구조가 필요했고 이런 문제를 파드가 해결한다. 파드를 통해 각 프로세스는 단일 컨테이너에서 실행되는 것처럼 동일한 환경을 제공받으면서 컨테이너로 격리된 상태를 유지할 수 있다.

### 파드에서 컨테이너를 격리하는 수준
쿠버네티스는 컨테이너 그룹을 관리하기 위해서 각 컨테이너가 전부는 아니지만 특정 리소스를 공유해서 완전히 격리되지 않도록 한다. 파드의 모든 컨테이너는 동일한 Network, IPC, UTS 네임스페이스에서 실행된다. 최신 버전에서는 PID 네임스페이스를 공유할 수도 있지만 기본적으로 활성화되어 있지 않다. Mount(파일시스템) 네임스페이스는 공유하지 않는다. 컨테이너는 이미지에서 파일시스템을 가져 오기 때문에 기본적으로 각 컨테이너의 파일시스템은 다른 컨테이너와 완전히 격리된다. 

파드의 컨테이너는 동일한 Network 네임스페이스에서 실행되기 때문에 동일한 IP, Port 공간을 공유한다. 또한 localhost를 통해 동일한 파드의 다른 컨테이너와 통신할 수 있다.

### 파드 간 네트워크 구성
쿠버네티스 클러스터의 모든 파드는 단일 평면 공유 네트워크 주소 공간에 있다. 모든 파드는 특정 파드가 어떤 노드에서 실행되는지에 상관 없이 파드에 할당된 IP를 통해서 서로 통신할 수 있다. LAN의 컴퓨터와 동일하게 각 파드는 파드 용으로 설정된 네트워크에서 고유한 IP를 할당받고 해당 IP를 통해서 다른 모든 파드에 접근할 수 있다.


![image](https://user-images.githubusercontent.com/44857109/105666335-5e416880-5f1c-11eb-9e61-bdf35e038032.png)

### 컨테이너 그룹화
파드는 비교적 가볍기 때문에 큰 오버 헤드 걱정없이 많은 수의 파드를 가질 수 있다. 따라서 밀접하게 관련된 구성 요소를 제외하고는 모든 것을 파드로 분리시키는 것이 좋다. 

프론트엔드, 백엔드로 구성된 다중 계층 어플리케이션을 파드로 구성하는 예를 생각해보자.

파드는 하나의 머신이라 생각해야 한다. 두 컨테이너가 동일한 파드로 구성된다면 항상 동일한 머신에서 실행된다. 2개의 워커 노드로 구성된 쿠버네티스 클러스터에서 1개의 단일 파드만 있는 경우 하나의 워커 노드는 파드를 할당받지 않기 때문에 리소스를 낭비하게 된다. 

또한 파드는 확장의 기본 단위이다. 쿠버네티스는 개별 컨테이너를 수평적으로 확장하는 대신 파드단위로 확장한다. 만약 2개의 컨테이터를 한 파드로 구성하면 백엔드의 인스턴스를 2개로 확장해야 할 때 프론트엔드도 함께 확장된다. 일반적으로 두 구성요소는 다른 확장 요구 사항이 있기 때문에 개별적으로 확장해야 한다. 따라서 두 컨테이너를 별도의 파드로 구성해야 한다. 

컨테이너를 파드로 그룹화할 때 다음을 고려하면 좋다

- 컨테이너들이 함께 실행되어야 하는가? 아니면 개별 머신에서 실행될 수 있는가?
- 컨테이너들이 서로 밀접하게 연관되어 있는가? 아니면 개별적인 구성 요소인가?
- 컨테이너들이 함께 확장되어야 하는가? 아니면 개별적으로 확장되어야 하는가?

## Creating Pods from YAML Descriptor
쿠버네티스 리소스는 일반적으로 Kubernetes REST API 엔드포인트에 매니페스트를 게시하여 생성된다. 일반적으로 리소스 정의는 다음과 같이 정의된다.

- apiVersion: 해당 디스크립터에 사용된 Kubernetes API 버전
- kind: 쿠버네티스 객체/리소스 유형
- metadata: 이름, 라벨, 주석 등의 메타데이터
- spec: 리소스 구성 요소
- status: 리소스의 세부 상태

매니페스트를 작성할 때 [쿠버네티스 참조 문서](http://kubernetes.io/docs/api) 또는 `kubectl explain` 명령어를 사용해서 API 개체에 대한 속성을 확인할 수 있다.

```
$ kubectl explain pods
$ kubectl explain pods.spec
```

### Descriptor를 통해 파드 생성

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: hostname-manual
spec:
  containers:
  - image: kbzjung359/kia-hostname
    name: hostname
    ports:
    - containerPort: 8080
      protocol: TCP
```

이 디스크립터는 Kubernetes API v1을 준수하고 이름이 hostname-manual인 Pod를 설정하고 있다. 이 파드는 kbzjung359/kia-hostname 이미지를 기반으로 하는 단일 컨테이너로 구성되고 8080 포트에서 수신 대기 중이다.

파드 정의에 포트를 지정하는 것은 정보 전달용이다. 포트 지정을 생략해도 특정 포트를 통해 파드에 접근할 수 있다. 하지만 클러스터를 사용하는 다른 사람이 파드가 노출하는 포트를 알 수 있도록 명시적으로 정의하는 것이 좋다. 

<br>

파드 생성은 다음 명령어를 사용한다.

```
$ kubectl create -f hostname-manual.yaml
pod/hostname-manual created
```

생성된 파드에 대한 매니페스트 정보를 확인할 수 있다. 

```
# pods -> po
$ kubectl get pods hostname-manual -o yaml
$ kubectl get pods hostname-manual -o json
```

<br>

컨테이너화 된 어플리케이션은 일반적으로 로그를 파일에 기록하는 대신 간단하고 표준적인 방법으로 확인할 수 있게 stdout, stderr에 기록한다. 파드의 로그를 확인하려면 다음 명령을 사용한다.

```
$ kubectl logs hostname-manual
server starting...
received request from ::ffff:172.17.0.1

# 파드에 컨테이너가 여러개일 때 특정한 컨테이너 지정
$ kubectl logs hostname-manual -c <container name>
```

컨테이너 로그는 매일 자동으로 교체되고 파일 크기가 10MB에 도달할 때마다 교체된다. `kubectl logs`는 가장 최근의 로그 항목만 표시한다. 파드가 삭제되면 해당 로그도 삭제된다. 파드가 삭제된 후에도 로그를 사용할 수 있도록 하려면 모든 로그를 중앙 저장소에 저장하는 방식으로 설정해야 한다.

<br>

생성한 파드에 접근해보자. `kubectl expose`를 사용해서 파드를 외부로 노출하는 서비스를 생성할 수 있지만 서비스를 사용하지 않고 디버깅 목적으로 파드에 연결하려 하는 경우에는 `kubectl port-forward`를 사용한다.

```
# 로컬 포트 8888를 파드 hostname-manual의 8080으로 전달
$ kubectl port-forward hostname-manual 8888:8080
```

## Organizing with Labels
실제 어플리케이션을 배포할 때 사용자는 많은 수의 파드를 실행하게 된다. 파드 수가 증가함에 따라 파드를 하위 집합으로 분류해야 할 필요성이 커진다. 임의의 기준에 따라 파드를 분류하면 개발자와 관리자는 파드가 어떤 것인지 쉽게 확인할 수 있고 각 파드에 대해서 개별적으로 작업을 수행할 필요없이 단일 잡업으로 특정 그룹의 모든 파드에서 작업할 수 있다. 이러한 작업을 지원하는 기능이 `Label`이다.

<br>

라벨은 파드뿐만 아니라 쿠버네티스의 모든 리소스를 구성하기 위한 기능이다. 라벨은 임의의 key-value 쌍으로 구성되고 key값이 중복되지 않는 상태에서 한 개 이상의 라벨을 설정할 수 있다. 일반적으로 리소스를 만들 때 라벨을 설정하지만 나중에 라벨을 추가하거나 기존 라벨을 수정할 수 있다.

![image](https://user-images.githubusercontent.com/44857109/105677652-7b336700-5f2f-11eb-98fd-a0d7788e3b87.png)

### 라벨과 함께 파드 생성
```yaml
# ...
metadata:
  # ...
  labels:
    creation_method: manual
    env: prod
#...
```

디스크립터의 메타데이터 절에 라벨 정보를 추가했다. 라벨은 `creation_method=manual`, `env=prod`를 추가했다. 파드의 라벨 정보를 확인해보자.

```
$ kubectl get pods --show-labels
NAME                   READY   STATUS    RESTARTS   AGE     LABELS
hostname-manual        1/1     Running   0          8m50s   creation_method=manual,env=prod
hostname-manual-beta   1/1     Running   0          11s     creation_method=manual,env=beta
```
 
모든 라벨을 출력하는 대신 -L 옵션으로 특정 라벨을 출력하도록 할 수 있다.

```
# env 라벨만 출력
$ kubectl get pods -L env
NAME                   READY   STATUS    RESTARTS   AGE     ENV
hostname-manual        1/1     Running   0          9m21s   prod
hostname-manual-beta   1/1     Running   0          42s     beta
```

### 기존 파드의 라벨 수정
파드에 새로운 라벨을 추가해보자

```
# hostname-manual 파드에 branch=main 라벨 추가
$ kubectl label pods hostname-manual branch=main
pod/hostname-manual labeled

$ kubectl get pods -L branch
NAME                   READY   STATUS    RESTARTS   AGE     BRANCH
hostname-manual        1/1     Running   0          16m     main
hostname-manual-beta   1/1     Running   0          7m44s   <none>
```

기존 라벨을 수정하려면 --overwrite 옵션을 사용해야 한다.

```
# hostname-manual 파드의 env 라벨을 prod -> debug 로 변경
$ kubectl label pods hostname-manual env=debug --overwrite
pod/hostname-manual labeled

$ kubectl get pods -L env
NAME                   READY   STATUS    RESTARTS   AGE     ENV
hostname-manual        1/1     Running   0          18m     debug
hostname-manual-beta   1/1     Running   0          9m37s   beta
```

### Label Selector를 통해 하위 집합 출력
라벨 셀렉터를 사용하면 특정 라벨로 태그가 지정된 파드의 하위 집합을 선택하고 해당 파드에서 작업을 수행할 수 있다. 필터링은 다음과 같은 기준으로 수행할 수 있다.

- 특정 키의 라벨을 포함하는지 또는 포함하지 않는지
- 특정 키와 값의 라벨이 있는지
- 특정 키의 라벨이 있지만 지정한 값과 같지 않는 것들

<br>

env 라벨이 debug로 설정된 리소스 출력
```
$ kubectl get pods -l env=beta
NAME                   READY   STATUS    RESTARTS   AGE
hostname-manual-beta   1/1     Running   0          20m
```

branch 라벨이 설정된 리소스 출력
```
$ kubectl get pods -l branch
NAME              READY   STATUS    RESTARTS   AGE
hostname-manual   1/1     Running   0          28m
```

branch 라벨이 없는 리소스 출력
```
$ kubectl get pods -l '!branch' --show-labels
NAME                   READY   STATUS    RESTARTS   AGE   LABELS
hostname-manual-beta   1/1     Running   0          22m   creation_method=manual,env=beta
```

필터링은 다양한 방식으로 수행할 수 있다.
- 'env!=beta' : env 라벨 값이 beta가 아닌 리소스
- 'env in (debug, beta) : env 라벨 값이 debug 또는 beta인 리소스
- 'env notin (debug) : env 라벨 값이 debug 이외의 값인 리소스
- 'env=beta,branch=main' : env, branch 라벨 값이 각각 beta, main인 리소스

## 파드의 노드 예약 제한
쿠버네티스는 클러스터의 모든 노드를 하나의 대규모 배포 플랫폼으로 노출하기 때문에 파드가 어떤 노드에 예약되는지는 중요하지 않다. 하지만 파드를 예약할 위치를 전달해야 하는 상황이 있을 수 있다. 만약 클러스터의 각 노드의 하드웨어 인프라가 동일하지 않은 경우 특정 파드를 SSD가 있는 노드에 예약되도록 하고 싶을 수 있다. 

파드가 어떤 노드에 예약되어야 하는지 구체적으로 지정하는 것은 바람직하지 않다. 대신 필요한 요구사항을 쿠버네티스에 전달하고 쿠버네티스가 해당 요구사항과 일치하는 노드를 선택하도록 해야한다. 이 작업은 노드 라벨, 노드 라벨 섹렉터를 통해 수행할 수 있다.

### 노드에 분류를 위한 라벨 설정
일반적으로 운영팀은 클러스터에 새 노드를 추가할 때 노드가 제공하는 하드웨어 정보 또는 파드를 예약할 때 유용할 수 있는 정보를 라벨로 지정한다. 

```
# minikube 노드에 `gpu=false`라벨 추가
$ kubectl label node minikube gpu=false
node/minikube labeled

# gpu=false 라벨이 있는 노드 출력
$ kubectl get nodes -l gpu=false
NAME       STATUS   ROLES                  AGE   VERSION
minikube   Ready    control-plane,master   25h   v1.20.2
```

### 특정 노드에 파드 예약
이제 GPU가 필요없는 파드를 배포해보자. 쿠버네티스 스케줄러에게 해당 정보를 전달하기 위해서 디스크립터에 노드 셀렉터 정보를 추가한다.

```yaml
# ...
spec:
  nodeSelector:
    gpu: "false"
  # ...
# ...
```

## Annotating
쿠버네티스 리소스들에는 라벨 외에도 주석을 달 수 있다. key-value 쌍 형식이기 때문에 라벨과 유사하지만 식별 정보를 의미하는 것은 아니다. 따라서 주석의 내용을 통해서 리소스들을 그룹화할 수 없다. 대신 주석은 라벨보다 훨씬 더 큰 정보를 포함할 수 있다. 주석의 가장 좋은 용도는 클러스터를 사용하는 모든 사람이 각 개별 객체에 대한 정보를 빠르게 찾을 수 있도록 설명을 추가하는 것이다. 

### 주석 조회
주석을 조회하려면 해당 객체의 디스크립터(yaml, json)을 요청하거나 `kubectl describe`를 사용해야 한다.

```
$ kubectl get pods hostname-manual -o yaml
...
metadata:
  annotations:
    ...
...

$ kubectl describe pods hostname-manual
...
Annotations: ...
...
```

### 주석 추가 및 수정
주석을 추가하는 가장 간단한 방법은 `kubectl annotate`를 사용하는 것이다.

```
$ kubectl annotate pod hostname-manual some.com/someannotation="some some"
```

## Using Namespace
쿠버네티스는 네임스페이스를 통해 객체들을 그룹화할 수 있다. 여러 네임스페이스를 사용하면 여러 구성 요소가 있는 복잡한 시스템을 더 작은 개별 그룹으로 분할할 수 있다. 

리소스 이름은 네임스페이스 내에서 고유해야 하지만 두 개의 서로 다른 네임스페이스에는 동일한 이름의 리소스가 있어도 상관 없다. 

### 네임스페이스 및 해당 포드 출력
먼저 네임스페이스를 출력해보자.

```
## namespace -> ns
$ kubectl get namespace
NAME                   STATUS   AGE
default                Active   30h
kube-node-lease        Active   30h
kube-public            Active   30h
kube-system            Active   30h
kubernetes-dashboard   Active   25h
```

`kubectl get`을 사용할 때 네임스페이스를 명시적으로 지정하지 않으면 항상 default 네임스페이스로 설정되어 해당 정보만 출력된다. 

kube-system 네임스페이스의 파드를 출력해보자.

```
# --namespace -> -n
$ kubectl get pods --namespace kube-system
NAME                               READY   STATUS    RESTARTS   AGE
coredns-74ff55c5b-wtxkz            1/1     Running   2          30h
etcd-minikube                      1/1     Running   2          30h
kube-apiserver-minikube            1/1     Running   2          30h
kube-controller-manager-minikube   1/1     Running   2          30h
kube-proxy-jwr4v                   1/1     Running   2          30h
kube-scheduler-minikube            1/1     Running   2          30h
storage-provisioner                1/1     Running   5          30h
```

네임스페이스를 사용하면 함께 속하지 않는 리소스를 겹치지 않는 그룹으로 분리할 수 있다. 여러 사용자가 동일한 쿠버네티스 클러스터를 사용하고 각각 고유한 리소스 집합을 관리하는 경우 각자 고유한 네임스페이스를 사용해야 한다. 이러면 다른 사용자의 리소스를 실수로 수정하거나 삭제하지 않도록 특벌한 주의를 기울일 필요가 없어지고 리소스 이름에 대한 충돌에 대해 걱정할 필요가 없어진다.

네임스페이스는 리소스를 결리하는 것 외에도 특정 사용자만 특정 리소스에 접근할 수 있도록 하고 개별 사용자가 사용할 수 있는 연산 리소스의 양을 제한하는 데에도 사용된다.

### 네임스페이스 생성
네임스페이스도 쿠버네티스의 리소스이므로 yaml파일을 Kubernetes API에 요청해서 만들 수 있다.

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: custom-namespace
```

파일을 작성하는 것이 귀찮다면 `kubernetes create namespace`를 사용해서 만들 수 있다. 

```
$ kubectl create namespace custom-namespace
namespace/custom-namespace created
```

### 네임스페이스에서 리소스 관리
custom-namespace 네임스페이스에 리소스를 생성하려면 매니페스트 파일의 metadata 섹션에 `namespace: custome-namespace`를 추가하거나 `kubectl create`에서 --namespace 옵션을 추가해야 한다.

```
$ kubectl create -f hostname-manual.yaml -n custom-namespace
```

특정 네임스페이스의 개체를 출력, 수정, 삭제할 때는 --namespace(-n) 플래그를 전달해야 한다. 해당 플래그를 지정하지 않으면 현재 kubectl 컨텍스트에 설정된 기본 네임스페이스에서 작업을 수행하기 때문에 원하지 않는 작업을 수행하게 될 수 있다. 컨텍스트는 `kubectl config`를 통해 변경할 수 있다.

### 네임스페이스를 통한 격리 수준
네임스페이스를 사용하면 지정된 네임스페이스에 속한 그룹에서만 작업할 수 있지만 실행중인 개체에 대한 어떤 종류의 격리도 제공하지 않는다. 예를 들어 서로 다른 네임스페이스에 파드를 배포하면 해당 파드가 서로 격리되어서 통신할 수 없다고 생각할 수 있지만 반드시 그런 것은 아니다. 네임스페이스가 격리를 제공하는지 여부는 쿠버네이스와 함께 배포되는 솔루션에 따라 다를 수 있다.

## Stopping and Removing Pods
`kubectl delete`를 통해 리소스를 삭제할 수 있다.

### 파드 이름으로 삭제
```
$ kubectl delete pods hostname-manual
pod "hostname-manual" deleted
```

파드를 삭제하도록 쿠버네티스에게 요청하면 쿠버네티스는 SIGTERM 시그널을 파드로 전달하고 정상적으로 종료될 때까지 특정 시간(기본적으로 30초) 동안 대기한다. 시간내에 종료되지 않으면 SIGKILL 시그널을 통해 강제종료시킨다. 따라서 컨테이너가 항상 정상적으로 종료되도록 하려면 SIGTERM 시그널을 정상적으로 처리하도록 설정해야 한다.

### 라벨 셀렉터로 파드 삭제
```
$ kubectl delete pods -l creation_method=manual
pod "hostname-manual" deleted
pod "hostname-manual-nogpu" deleted
```

### 전체 네임스페이스를 삭제해서 파드 삭제
네임스페이스를 삭제하면 파드는 함께 자동으로 삭제된다.

```
$ kubectl delete ns custom-namespace
```

### 네임스페이스는 유지하면서 네임스페이스의 모든 파드 삭제
--all(-A) 옵션을 사용해서 현재 네임스페이스의 모든 파드를 삭제할 수 있다.

```
$ kubectl delete pods --all
```

### 네임스페이스의 모든 리소스 삭제
단일 명령으로 현재 네임스페이스의 모든 리소스(컨트롤러, 파드, 서비스 등)를 삭제할 수 있다.

```
$ kubectl delete all --all
```

명령어에서 첫번째 all은 모든 유형의 리소스를 삭제하도록 지정하고 --all 옵션은 모든 리소스 인스턴스를 삭제하도록 지정한다.