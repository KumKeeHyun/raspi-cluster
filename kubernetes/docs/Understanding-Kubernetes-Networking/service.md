# Service Network
`pod network`를 통해서 서로 다른 노드에서 실행중인 파드가 통신할 수 있다. 하지만 파드는 일시적으로 존재하고 다시 생성될 때 같은 주소를 갖는다는 보장이 없다. 따라서 파드의 주소를 엔드포인트로 사용하기엔 무리가 있다. 
<br>

이 문제에는 `리버스 프록시`, `로드 밸런서`를 이용한 표준 솔루션이 있었다. 이 방법에는 몇가지 요구사항이 있었다.

- 프록시는 자체적으로 내구성이 있어야 함. 
- 클라이언트의 요청을 전달할 서버의 목록을 유지해야 함.
- 어떤 서버가 요청을 처리할 수 있는 정상 상태인지 알 수 있는 방법이 있어야 함.

`kubernetes service`는 이러한 요구사항을 구현한 리소스 타입이다. 

## Example Deployment State
|type|name|ip|port|network|
|:-:|:-:|:-:|:-:|:-:|
|pod|server1|10.0.1.2|8080|10.0.0.0/14|
|pod|server2|10.0.2.2|8080|10.0.0.0/14|
|pod|client|10.0.1.3|x|10.0.0.0/14|
|service|server-service|10.3.241.152|80|10.3.240.0/20|
|host|host1|10.100.0.2|x|10.100.0.0/24|
|host|host2|10.100.0.3|x|10.100.0.0/24|

<img src="https://user-images.githubusercontent.com/44857109/103441209-b1aef680-4c8f-11eb-81d7-3ee90aad7b2b.png" width="80%" height="80%">

## Service Routing

서비스 네트워크와 파드 네트워크는 모두 가상이지만 몇가지 차이점이 있다. 클러스터를 구성하는 노드(호스트)에서 네트워크 브리지, 인터페이스를 나열(`ifconfig`)해보면 파드 네트워크는 확인할 수 있지만 서비스 네트워크는 볼 수 없다. 서비스 네트워크는 실제로 존재하지 않고 어떠한 인터페이스에도 연결되어 있지 않다. 하지만 서비스 네트워크의 IP 주소를 통해 특정 파드에 접근할 수 있다.
<br>

라우팅 시나리오는 다음과 같다. `client`는 DNS를 이용해서 `server-service`로 http 요청을 보낸다. 쿠버네티스 DNS를 통해 `10.3.241.152`로 IP 주소를 확인하고 `client`는 해당 주소로 패킷을 전송하게 된다. 

<img src="https://user-images.githubusercontent.com/44857109/103441655-c8a31800-4c92-11eb-868d-21e6d2f2db92.png" width="80%" height="80%">

`veth0`, `cbr0`은 `10.3.241.152`장치를 확인할 수 없기 때문에 업스트림 인터페이스로 패킷을 전달한다. `eth0` 인터페이스 또한 `10.100.0.0/24` 네트워크에 있기 때문에 패킷을 게이트웨이로 전달한다. 대신 이때 패킷은 서비스에서 적절한 파드로 리다리렉션되어서 전달된다. 이러한 리다이렉션을 처리해주는 것이 `kube-proxy`이다.

## Kube Proxy
### problem 1
프록시의 일반적인 동작은 두 개의 커넥션을 통해 서버와 클라이언트간 트래픽을 전달하는 것이다. 이때 프록시는 `user space`에서 실행되기 때문에 패킷은 프록시를 통과할 때마다 `user space`와 `kernel space`를 오간다. `kube-proxy`도 처음엔 유저 공간으로 구현되었지만 약간 수정되었다.
<br>

### problem 2
프록시는 클라이언트의 연결을 수신하고 서버에 연결하는데 사용할 네트워크 인터페이스가 필요하다. 노드에서 사용할 수 있는 인터페이스는 `호스트의 인터페이스`, `파드 네트워크의 가상 인터페이스` 2가지가 있지만 서비스 네트워크에는 어떠한 인터페이스에도 연결되어 있지 않다. 때문에 라우팅 규칙이나 방화벽 필터 등을 사용할 수 없다.
<br>

`kubernetes`는 이 문제를 Linux 커널의 기능인 `netfilter`와 유저 공간 인터페이스인 `iptables`을 통해 해결한다. netfilter는 일종의 커널 공간 프록시다. 때문에 커널 공간에서 패킷을 검사하고 특정한 조치를 취할 수 있다.

### redirect to host
`kube-proxy`는 호스트의 인터페이스(10.100.0.2)에서 특정 포트(예시에선 10400)를 열어 `server-serivce` 서비스에 대한 요청을 수신한다. 이후 `netfilter`에게 서비스 IP에 대한 패킷을 `10.100.0.2:10400`으로 재라우팅하도록 설정한다. 이후 `10.3.241.152:80`으로 요청하는 패킷은 netfilter에 의해 kube-proxy로 라우팅되고 kube-proxy는 이 요청을 `10.0.2.2:8080`으로 포워딩한다.

<img src="https://user-images.githubusercontent.com/44857109/103442721-343db300-4c9c-11eb-91a0-c5854fb92998.png" width="50%" height="50%">


### redirect to pod
이전의 방법은 kube-proxy가 유저공간에서 실행된다. kubernetes 1.2 버전 이후에는 `iptables` 기능을 통해 서비스 IP를 특정 파드로 포워딩하는 역할을 `netfilter`로 위임하게 되었다. 이 방식에서 kube-proxy의 역할은 netfilter rule을 클러스터 파드 정보와 동기화하는 것이다.

<img src="https://user-images.githubusercontent.com/44857109/103442797-ae6e3780-4c9c-11eb-9002-eb1d0c4645b3.png" width="50%" height="50%">