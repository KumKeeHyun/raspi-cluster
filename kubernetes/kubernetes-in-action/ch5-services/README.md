# Chapter 5. Services: enabling clients to discover and talk to pods

<!--ts-->
  - [5.1 Introducing Services](#5.1-Introducing-Services)
    - [서비스 생성](#서비스-생성)
    - [클러스터 내에서 서비스로 접근](#클러스터-내에서-서비스로-접근)
    - [서비스 리소스 사용](#서비스-리소스-사용)
  - [5.2 Connecting to Services Living Outside the Cluster](#5.2-Connecting-to-Services-Living-Outside-the-Cluster)
    - [엔드포인트](#엔드포인트)
    - [수동으로 엔드포인트 구성](#수동으로-엔드포인트-구성)
  - [5.3 Exposing Services to External Clients](#5.3-Exposing-Services-to-External-Clients)
    - [NodePort](#NodePort)
    - [LoadBalancer](#LoadBalancer)
    - [외부 연결의 특성 이해](#외부-연결의-특성-이해)
  - [5.4 Exposing Services Externally Through an Ingress Resource](#5.4-Exposing-Services-Externally-Through-an-Ingress-Resource)
    - [Ingress 생성](#Ingress-생성)
    - [Ingress 작동 방식](#Ingress-작동-방식)
    - [Ingress에 여러 서비스 노출시키기](#Ingress에-여러-서비스-노출시키기)
  - [](#)
  - [](#)
<!--te-->

## 5.1 Introducing Services
파드에서 실행되는 서비스를 사용하려는 경우 클라이언트 서비스는 원하는 서비스가 실행되고 있는 파드를 찾는 방법이 필요하다. 쿠버네티스를 사용하지 않는다면 클라이언트 구성 파일에 서버의 정확한 IP 주소를 지정하는 방식으로 진행하지만 쿠버네티스는 그렇게 작업하기엔 문제가 있다.

- 파드는 임시적인 리소스이다. 파드는 배포 정책 또는 장애에 따라 수시로 생성, 삭제되기 때문에 파드의 IP를 직접적으로 사용하는 것은 적절하지 않다.
- 파느의 IP는 클러스터에서 파드가 스케쥴되고 시작하기 전에 부여되기 때문에 클라이언트는 서버 파드의 IP를 미리 알 수 없다.
- 파드는 수평적으로 확장되기 때문에 동일한 서비스를 제공하는 여러개의 파드 IP가 존재한다. 클라이언트는 파드가 얼마나 많이 존재하는지, 해당 파드들의 IP가 무엇인지 다룰 수 없다. 클라이언트는 이 파드들에 접근하기 위한 하나의 IP만 알아야한다.

쿠버네티스의 Service는 동일한 서비스를 제공하는 파드 그룹에 대한 단일 진입점을 만들기 위한 리소스이다. 각 Service에는 해당 리소스가 존재하는 동안 변경되지 않는 IP 및 포트가 있다. 클라이언트는 해당 IP 및 포트를 통해 서비스에 접근하고 이 요청은 Service를 지원하는 파드중 하나로 라우팅된다.

아래 그림은 서비스의 사용 사례를 보여준다.

![image](https://user-images.githubusercontent.com/44857109/106131217-bfc83800-61a5-11eb-92ab-c2cfdb888ef3.png)

### 서비스 생성
서비스로 들어오는 요청은 해당 서비스가 관리하는 파드로 로드밸런싱된다. 서비스에 포함되는 파드와 그렇지 않는 파드를 정의하는 방법은 라벨 셀렉터와 동일한 메커니즘을 사용한다.

![image](https://user-images.githubusercontent.com/44857109/106132321-2863e480-61a7-11eb-86f9-c4c5b4f06bc2.png)

서비스를 생성하는 가장 쉬운 방법은 이미 만들어져 있는 컨트롤러 리소스를 `kubectl expose`로 노출시키는 것이다. 이 명령어는 해당 컨트롤러에서 사용하는 것과 동일한 파드 셀렉터로 서비스 리소스를 생성하고 단일 IP 및 포트를 통해 파드를 외부로 노출한다.

이전 챕터와 동일하게 YAML 디스크립터로 생성해보자.

```yaml
apiVersion: v1
kind: Service
metadata:
  name: hostname
spec:
  ports:
  - port: 80 # 이 서비스는 80번 포트를 
    targetPort: 8080 # 파드의 8080번 포트로 리다이렉트한다.
  selector:
    app: hostname # app=hostname 라벨이 있는 파드들에 대해서
```

```
$ kubectl get svc
NAME         TYPE        CLUSTER-IP       EXTERNAL-IP   PORT(S)   AGE
hostname     ClusterIP   10.108.167.231   <none>        80/TCP    4s
kubernetes   ClusterIP   10.96.0.1        <none>        443/TCP   4d4h

$ 
```

생성한 서비스에 할당된 IP는 `10.108.167.231`이다. 이 IP는 ClusterIP이기 때문에 클러스터 내에서만 사용할 수 있다. 서비스의 주요 목적은 파드 그룹을 클러스터의 다른 파드에 노출하는 것이다. 하지만 일반적으로 서비스를 클러스터 외부로 노출시키는 작업도 수행한다. 일단은 클러스터 내부에서 서비스가 어떤 역할을 하는지 알아보자.

<br>

### 클러스터 내에서 서비스로 접근
클러스터 내에서 서비스에 요청을 보내는 방법은 다음과 같다.

- 서비스의 ClusterIP로 요청을 보내고 응답을 기록하는 파드를 생성
- `ssh`를 이용해서 쿠버네티스 클러스터 누드 중 하나로 접속해서 ClusterIP로 요청
- `kubectl exec`으로 기존 파드 중 하나에서 ClusterIP로 요청하도록 수행

`kubectl exec`을 사용하면 파드의 기존 컨테이너 내에서 임의의 명령어를 원격으로 실행할 수 있다. 보통 컨테이너의 상태 및 환경을 검사할 때 사용한다. 이 명령어를 통해서 기존 파드에서 ClusterIP로 요청을 보내보자.

```
$ kubectl exec hostname-rc-qsth7 -- curl -s http://10.108.167.231:80
You've hit hostname-rc-9qhbv
```

> 파드에서 원격으로 실행할 명령어 인자를 `--`을 사용해서 전달했다. 원격으로 실행할 명령에 대시로 시작하는 인수가 있는경우 kubectl은 해당 인자를 원격 명령이 아니라 자신의 옵션으로 해석해서 오류가 발생한다. 따라서 이중 대시를 사용해서 kubectl 명령과 원격 명령을 명시적으로 구분하는 것이 좋다.

![image](https://user-images.githubusercontent.com/44857109/106137479-1c2f5580-61ae-11eb-840d-1f3b9ddaf7c9.png)

<br>

#### session affinity 구성
서비스 프록시는 일반적으로 연결이 동일한 클라이언트에서 오는 경우에도 요청을 무작위 파드에 전달하기 때문에 동일한 요청을 보내도 각각 다른 파드에서 응답이 올 수 있다. 특정한 클라이언트의 모든 요청을 매번 동일한 파드로 리다이렉션하려면 `sessionAffinity` 속성을 이용해야 한다.

```yaml
apiVersion: v1
kind: Service
# ...
spec:
  sessionAffinity: ClientIP
# ...
```

이렇게하면 서비스 프록시가 동일한 클라이언트에서 온 모든 요청을 동일한 파드로 전달한다. 해당 옵션은 `None`, `ClientIP` 두가지 밖에 없다. 쿠키 기반의 세션 옵션이 없는 이유는 쿠버네티스 서비스 리소스가 HTTP 수준에서 작동하지 않기 때문이다. 

<br>

#### 한 서비스에 여러 포트 노출
서비스는 여러 포트를 노출시킬 수 있다. 즉 파드가 8080, 8443 2개의 포트를 사용하는 경우에도 한 서비스를 사용해서 구성할 수 있다. 여러 포트가 있는 서비스를 만들 때 디스크립터에 각 포트의 이름을 지정해야 한다. 

```yaml
apiVersion: v1
kind: Service
# ...
spec:
  ports:
  - name: http # 80번 포트의 이름은 http
    port: 80
    targetPort: 8080
  - name: https # 433번 포트의 이름은 https
    port: 443
    targetPort: 8443
# ...
```

이렇게 서비스를 구성하면 라벨 셀렉터에 의해 구분된 파드들에서 8080, 8443 포트를 노출시킬 수 있다. 위에서는 노출할 파드의 포트를 번호로 지정했지만 만약 파드를 생성할 때 포트에 이름을 지정했다면 해당 포트 이름을 이용해서 지정할 수도 있다.

```yaml
kind: Service
# ...
spec:
  containers:
  - name: hostname
    ports:
    - name: http
      containerPort: 8080
    - name: https
      containerPort: 8443
# ...
```

```yaml
kind: Service
# ...
spec:
  ports:
  - name: http
    port: 80
    targetPort: http # 파드에서 http라는 이름으로 지정한 포트 번호 -> 8080
  - name: https
    port: 443
    targetPort: https # 파드에서 https라는 이름으로 지정한 포트 번호 -> 8443
# ...
```

파드의 포트 이름을 통해 서비스를 구성하면 나중에 파드가 사용하는 포트 번호가 변경될 때 서비스를 수정하지 않아도 된다는 것이다.

### 서비스 리소스 사용
서비스를 생성하면 서비스가 관리하는 파드들에 접근할 수 있는 고정된 단일 IP 및 포트를 갖게 된다. 그러면 클라이언트 파드는 해당 서비스의 IP 및 포트를 어떻게 알 수 있을까? 서비스를 생성한 뒤에 할당된 IP를 확인하고 클라이언트 파드에게 설정 정보를 전달하는 것은 그다지 좋아보이지 않는다. 쿠버네티스는 파드가 서비스의 IP 및 포트를 검색하는 방법을 제공한다.

<br>

#### 환경변수
파드가 시작되면 쿠버네티스는 해당 시점에 존재하는 각 서비스 정보를 환경변수로 초기화해준다. 한번 확인해보자.

```
$ kubectl exec hostname-rc-qsth7 -- env | grep SERVICE
KUBERNETES_SERVICE_PORT=443
KUBERNETES_SERVICE_PORT_HTTPS=443
KUBERNETES_SERVICE_HOST=10.96.0.1
HOSTNAME_SERVICE_PORT=80
HOSTNAME_SERVICE_HOST=10.108.167.231
```

> 환경변수 이름으로 표시될 때 서비스 이름의 대시는 밑줄로 변환되고 모든 문자가 대문자로 표시된다.

파드가 이미 실행되고 있는 상태에서 서비스를 만들었다면 해당 서비스에 대한 정보가 초기화되어있지 않기 때문에 파드를 지우고 다시 생성해야 한다. 

<br>

#### DNS
kube-system 네임스페이스에 있는 파드를 출력해보면 `coredns` 또는 `kube-dns`가 있다. 이 서비스는 클러스터에서 실행중인 모든 파드가 자동으로 사용하도록 구성되는 DNS 서버이다. 

> 쿠버네티스는 각 컨테이너의 `/etc/resolv.conf` 파일을 수정해서 쿠버네티스 내부에서 실행하는 DNS 서버를 설정한다. 

클라이언트 파드는 환경변수에 의존하는 대신 FQDN(정규화된 도메인 이름 형식)을 통해 서비스에 접근할 수 있다.

<br>

쿠버네티스 클러스터 내부에서 사용하는 로컬 서비스의 FQDN 형식은 다음과 같다.

```
{service-name}.{namespace}.svc.cluster.local

# hostname 서비스의 경우
hostname.default.svc.cluster.local
```

만약 클라이언트 파드가 서버 파드와 동일한 네임스페이스에 있다면 `svc.cluster.local` 접미사와 네임스페이스를 생략할 수 있다. 하지만 여전히 클라이언트 파드는 서버 서비스의 포트 번호를 알고있어야 한다. MySQL, HTTP 같은 표준 포트를 사용하는 경우엔 상관 없지만 그렇지 않은 경우엔 환경변수를 통해 서비스 포트 번호를 알아내야 한다.

```
$ kubectl hostname-rc-9qhbv -- curl -s hostname:80
You've hit hostname-rc-qsth7

$ kubectl hostname-rc-9qhbv -- curl -s hostname.default.svc.cluster.local:80
You've hit hostname-rc-qsth7

$ kubectl exec -it hostname-rc-9qhbv -- bash
# curl hostname:80
You've hit hostname-rc-qsth7
```

<br>

## 5.2 Connecting to Services Living Outside the Cluster
쿠버네티스 서비스 리소스는 클라이언트 파드가 내부 서비스에 접근하는 것처럼 클러스터 외부에 있는 서비스에 접근할 수 있도록 구성할 수 있다.

### 엔드포인트
사실 서비스는 파드에 직접적으로 연결되지 않는다. 즉 서비스의 파드 셀렉터는 서비스로 들어오는 요청을 리다이렉션할 때 직접 사용되지 않는다. 대신 파드 셀렉터를 사용하여 얻은 파드들의 IP 및 포트 목록을 Endpoint 리소스에 저장한다. 클라이언트가 서비스에 요청을 보내면 서비스 프록시는 Endpoint에 있는 목록 중 하나를 선택해서 리다이렉션한다.

`kubectl describ svc` 또는 `kubectl get endpoints`를 통해 실제 서비스가 유지하고 있는 Endpoint를 확인할 수 있다.

```
$ kubectl decribe svc hostname
Name:              hostname
Namespace:         default
Labels:            <none>
Annotations:       <none>
Selector:          app=hostname
Type:              ClusterIP
IP Families:       <none>
IP:                10.108.167.231
IPs:               10.108.167.231
Port:              <unset>  80/TCP
TargetPort:        8080/TCP
Endpoints:         172.17.0.2:8080,172.17.0.3:8080
Session Affinity:  None

$ kubectl get endpoints hostname
NAME       ENDPOINTS                         AGE
hostname   172.17.0.2:8080,172.17.0.3:8080   18h
```

### 수동으로 엔드포인트 구성
파드 셀렉터를 지정하지 않고 서비스를 생성하면 쿠버네티스는 엔드포인트 리소스를 만들지 않는다. 이런 경우 서비스에 대한 엔드포인트 목록을 사용자가 수동으로 지정할 수 있다.

```yaml
apiVersion: v1
kind: Service
metadata:
  name: external-service
spec:
  ports:
  - port: 80
```

```yaml
apiVersion: v1
kind: Endpoints
metadata:
  name: external-service
subsets:
  - addresses:
    - ip: 11.11.11.11
    - ip: 22.22.22.22
    ports:
    - port: 80
```

서비스를 파드 셀렉터 없이 생성했기 때문에 서비스에 대한 엔트포인트 리소스를 수동으로 만들어줘야 한다. 이때 주의할 점은 서비스와 엔트포인트 이름을 동일하게 설정해야 하는 것이다. 이 구성에 대한 그림은 아래와 같다.

![image](https://user-images.githubusercontent.com/44857109/106240153-205b8180-6247-11eb-9f61-1b7217e9ea9b.png)

나중에 외부 서비스를 쿠버네티스에서 실행되는 파드로 마이그레이션하기로 결정했다면 파드의 구성을 변경할 필요 없이 해당 서비스에 파드 셀렉터를 추가해서 엔드포인트가 자동으로 관리되도록 할 수 있다. 역순으로도 쉽게 마이그래이션할 수 있다.  

<br>

도메인을 사용한다면 외부 엔트포인트를 더 간단하게 구성할 수 있다. 예를 들어 외부에 `api.somecompany.com`로 사용가능한 API가 있다고 한다면 `ExternalName`을 이용해서 서비스를 구성할 수 있다.

```yaml
apiVersion: v1
kind: Service
metadata:
  name: external-service
spec:
  type: ExternalName
  externalName: api.somecompany.com
  ports:
  - port: 80
```

서비스가 생성되면 클러스터 내부의 파드는 해당 외부서비스에 `api.somecompany.com` 뿐만 아니라 `external-service.default.svc.cluster.local`으로도 접근할 수 있게 된다. 이렇게 하면 파드에서 실제 서비스 이름과 해당 위치가 숨겨지기 때문에 나중에 언제든지 서비스 구성을 수정해서 다른 외부 서비스를 가리킬 수 있다. 

ExternalName은 DNS 수준에서만 사용할 수 있다. 또한 ClusterIP가 생성되지 않기 때문에 서비스 프록시를 사용하지 않고 외부서비스로 직접 연결된다.

<br>

## 5.3 Exposing Services to External Clients
서비스 리소스의 타입을 지정하지 않고 생성하면 기본 타입은 ClusterIP이다. 해당 서비스는 클러스터 내부에서만 사용할 수 있지만 `NodePort`, `LoadBalancer` 타입 또는 `Ingress` 리소스를 통해 클러스터 내부의 파드를 외부로 노출할 수 있다.

![image](https://user-images.githubusercontent.com/44857109/106244797-d2e31280-624e-11eb-8849-5424e45edf95.png)

- NodePort: 클러스터 내의 모든 노드에서 특정 포트를 열고 해당 포트로 들어오는 요청을 서비스로 리다이렉션한다. 서비스는 클러스터 내부에서 ClusterIP로 접근할 수 있을뿐만 아니라 모든 노드의 IP를 통해서도 접근할 수 있다.
- LoadBalancer: NodePort를 확장하는 타입이다. 클라우드 플랫폼에서 프로비저닝해주는 로드밸런서를 통해서 해당 서비스에 접근할 수 있게 해준다.
- Ingress: 여러 개의 서비스를 한 IP로 노출시킬 수 있는 다른 타입의 리소스이다. HTTP 수준에서 작동되기 떄문에 서비스보다 더 많은 기능을 제공한다.

### NodePort
NodePort 타입으로 서비스를 생성해보자.

```yaml
apiVersion: v1
kind: Service
metadata:
  name: hostname-nodeport
spec:
  type: NodePort
  ports:
  - port: 80
    targetPort: 8080
    nodePort: 30123
  selector:
    app: hostname
```

이전에 생성했던 서비스에서 `kubectl edit`을 통해 NodePort로 수정했다.

```
$ kubectl get svc
NAME         TYPE        CLUSTER-IP       EXTERNAL-IP   PORT(S)        AGE
hostname     NodePort    10.108.167.231   <none>        80:30123/TCP   20h
kubernetes   ClusterIP   10.96.0.1        <none>        443/TCP        5d
```

PORT가 ClusterIP 포트와 NodePort를 나타내는 `80:30123`으로 변경되었다. 이제 해당 서비스에 다음과 같이 접근할 수 있다.

- 10.108.167.231:80
- {Node's IP}:30123

![image](https://user-images.githubusercontent.com/44857109/106248312-27d55780-6254-11eb-9eb3-95aa19aa5448.png)

Node1으로 들어온 요청은 서비스로 리다이렉션되기 때문에 Node2에서 실행되고 있는 파드로 전달될 수도 있다. 또한 모든 노드의 포트 30123을 통해 서비스에 접근할 수 있기 때문에 클라이언트가 어떤 노드에 요청을 보내야 할지는 중요하지 않다. 하지만 클라이언트가 하나의 노드로만 요청을 하면 해당 노드가 실패할 때 클라이언트는 해당 서비스에 접근할 수 없게된다. 따라서 NodePort를 사용할 땐 노드 앞에 로드밸런서를 배치해서 모든 정상 노드에 요청을 분산하는 것이 적절하다.

### LoadBalancer
이 타입은 GKE 같은 클라우드 서비스에서 사용하는 것 같아서 패스했다.

### 외부 연결의 특성 이해
서비스를 외부로 노출할 때 몇가지 고려할 사항이 있다.

#### 불필요한 네트워크 홉 방지
외부 클라이언트가 NodePort를 통해 서비스에 연결하는 경우 임의로 선택된 파드가 연결을 수신한 노드에서 실행되는 파드가 아닌 경우 네트워크 홉이 증가하게 된다. 

`externalTrafficPolicy` 속성을 이용해서 외부 요청을 수신한 노드에서 실행중인 파드로만 트래픽을 전달하는 서비스를 구성할 수 있다. 하지만 요청을 수신한 노드에 해당 서비스의 파드가 실행중이지 않으면 연결은 중단된다. 따라서 로드밸런서가 해당 파드가 하나 이상 있는 노드에만 트래픽을 전달하는지 확인해야 한다.

```yaml
apiVersion: v1
kind: Service
# ...
spec:
  externalTrafficPolicy: Local
# ...
```

이 옵션이 항상 최선의 방법이 되는 건 아니다. 일반적으로 externalTrafficPolicy 속성을 사용하면 트래픽이 모든 파드에 균등하게 분산되지 않는다. 

![image](https://user-images.githubusercontent.com/44857109/106253245-96b5af00-625a-11eb-8309-2ed117d70d94.png)


#### 클라이언트의 IP가 보존되지 않음
일반적으로 클러스터 내부의 클라이언트가 서비스에 요청하면 해당 서비스의 파드는 클라이언트의 IP를 얻을 수 있다. 하지만 NodePort를 통해 들어오는 외부 요청에 대해서는 패킷에 SNAT(Source Network Address Translation)가 수행되기 때문에 실제 클라이언트 IP를 볼 수 없다.  

externalTrafficPolicy 속성을 로컬로 설정한 서비스에 대해서는 추가적인 네트워크 홉이 발생하기 않기 때문에 이 경우에는 클라이언트 IP가 보존된다.

## 5.4 Exposing Services Externally Through an Ingress Resource
서비스를 외부로 노출시키는 경우 각 서비스당 하나의 외부 IP와 로드밸런서를 사용해야 한다. Ingress 리소스는 호스트 또는 경로에 따라 요청을 여러 서비스에 포워딩하기 때문에 하나의 IP로 여러 서비스에 접근할 수 있게 해준다. 또한 Ingress는 7계층에서 작동하기 때문에 HTTP에 관련된 다양한 기능(Session Affinity, TLS)을 수행할 수 있다.

### Ingress 생성
YAML 디스크립터로 Ingress를 생성해보자.

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: hostname
spec:
  rules:
  - host: hostname.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: hostname-nodeport
            port:
              number: 80
```

이 Ingress는 Ingress Controller에서 호스트가 `hostname.example.com`인 수신한 모든 HTTP 요청을 hostname-nodeport 서비스로 포워딩한다.

```
$ kubectl get ingress
NAME       CLASS    HOSTS                  ADDRESS        PORTS   AGE
hostname   <none>   hostname.example.com   192.168.49.2   80      2m23s
```

이제 `hostname.example.com`을 `192.168.49.2`으로 인식할 수 있도록 DNS 서버를 구성하거나 `/etc/hosts`에 다음을 추가하자.

```
# /etc/hosts에 추가
192.168.49.2    hostname.example.com
```

이제 컴퓨터에서 hostname.example.com 도메인을 Ingress Controller로 인식할 수 있게 되었다. 이 도메인으로 요청을 보내보자.

```
$ curl hostname.example.com
You've hit hostname-rc-9qhbv
```

### Ingress 작동 방식
![image](https://user-images.githubusercontent.com/44857109/106282395-df359280-6283-11eb-8c14-437089c9be3f.png)

Ingress Controller는 수신한 HTTP 요청의 헤더를 보고 `hostname.example.com` 도메인이 어떤 서비스로 포워딩해야 하는지 판단한다. 이후 요청을 해당 서비스로 포워딩하지 않고 서비스의 엔드포인트를 조회해서 얻은 파드 리스트중 하나를 선택해서 직접 파드로 포워딩한다.

### Ingress에 여러 서비스 노출시키기
Ingress 디스크립터에서 `rules`, `paths`는 배열이기 때문에 여러개의 호스트, 경로를 지정할 수 있다. 먼저 경로를 여러개로 설정해보자.

```yaml
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: hostname
spec:
  rules:
  - host: hostname.example.com
    http:
      paths:
      - path: /first
        backend:
          serviceName: hostname-nodeport
          servicePort: 80 
      - path: /second
        backend:
          serviceName: hostname2-nodeport
          servicePort: 80
```

이 경우 요청청된 URL 경로에 따라 다른 서비스로 요청이 포워딩된다.

```
# hostname-nodeport 서비스로 포워딩
$ curl hostname.example.com/first

# hostname2-nodeport 서비스로 포워딩
$ curl hostname.example.com/second
```

> Ingress의 API 버전이 networking.k8s.io/v1가 되면서 path 설정 방법이 달라졌다. 그래서 위 방식은 extensions/v1beta1 버전에서만 가능한 것 같다.

<br>

이제 호스트를 여러개로 구성해서 Ingress를 생성해보자.

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: hostname
spec:
  rules:
  - host: hostname.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: hostname-nodeport
            port:
              number: 80
  - host: example.hostname.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: hostname2-nodeport
            port:
              number: 80
```

```
# hostname-nodeport 서비스로 포워딩
$ curl example.hostname.com

# hostname2-nodeport 서비스로 포워딩
$ curl hostname.example.com
```


