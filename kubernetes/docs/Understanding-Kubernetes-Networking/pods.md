# Pods Networking

## Docker Container
물리적 네트워크 `eth0`에 네트워크 가상화를 위해 브리지 `docker0`을 연결하고, 컨테이너는 가상 네트워크 인터페이스 `veth0`, `veth1` 등으로 브리지에 연결된다. 

<img src="https://user-images.githubusercontent.com/44857109/103437591-c037e680-4c6c-11eb-862b-a7133262f14a.png" width="50%" height="50%">

## Shared network stack
한 `kubernetes pod`는 한 개 또는 여려 개의 컨테이너를 실행할 수 있고 각 컨테이너는 네트워크 스택을 공유한다. 네임스페이스 격리를 약간 수정해서 각 컨테이너가 네트워크를 공유하도록 만들 수 있다.

<img src="https://user-images.githubusercontent.com/44857109/103437685-a77c0080-4c6d-11eb-8f9c-d27ab58d798c.png" width="50%" height="50%">


## Pods
개별 `kubernetes node`에서 `docker ps`를 실행해보면 `pause`로 시작하는 컨테이너가 존재한다. `pause`는 쿠버네티스로부터 `SIGTERM` 시그널을 받으면 실행중인 컨테이너를 중지시키는 역할을 한다. 또한 컨테이너들이 외부와 통신할 수 있도록 가상 네트워크 인터페이스를 제공하기 때문에 `kubernetes pod`의 심장이라 볼 수 있다.
<br>

개별 포드에 대한 그림은 다음과 같다.

<img src="https://user-images.githubusercontent.com/44857109/103437890-283bfc00-4c70-11eb-8451-d7aea22ec6df.png" width="50%" height="50%">

## Pod Network
`kubernetes cluster`는 한개 이상의 노드로 이루어 진다. 클러스터에 대한 그림은 다음과 같다.

<img src="https://user-images.githubusercontent.com/44857109/103438110-b913d700-4c72-11eb-9f7e-bdb98adff262.png" width="80%" height="80%">

이 경우 `172.17.0.1`가 어떤 호스트의 브리지인지 알기 어렵다. 패킷은 각 노드의 브리지에 할당된 주소를 구별해야 한다. `kubernetes`는 두 가지 방식을 통해 이 문제를 해결한다. 먼저 `각 노드의 브리지에 대한 전체 주소 공간`을 할당한 다음 각 노드에서 해당 주소 공간에 속하도록 브리지를 생성한다. 다음으로 게이트웨이에 `할당된 브리지에 대한 라우팅 테이블 규칙`을 추가해서 패킷이 노드내의 브리지로 향할 수 있도록 한다. 쿠버네티스에서는 일반적으로 `pod network(overlay network)`라고 부른다. 
<br>

이 네트워크에 대한 그림은 다음과 같다.

<img src="https://user-images.githubusercontent.com/44857109/103438816-abfae600-4c7a-11eb-9e16-65f4e85cda80.png" width="80%" height="80%">

이렇게 되면 `10.0.1.2`에 대한 패킷에 대해서 게이트웨이는 host1에게 라우팅할 수 있게 된다. 