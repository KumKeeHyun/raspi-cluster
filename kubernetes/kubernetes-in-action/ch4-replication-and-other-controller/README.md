# 4. Replication and Other Controller

<!--ts-->
  - [4.1 Keeping Pods Healthy](#4.1-Keeping-Pods-Healthy)
    - [활성 프로브](#활성-프로브)
    - [HTTP GET Probe](#HTTP-GET-Probe)
    - [활성 프로브의 추가 변수](#활성-프로브의-추가-변수)
    - [좋은 활성 프로브](#좋은-활성-프로브)
  - [4.2 ReplicationController](#4.2-ReplicationController)
    - [ReplicationController 생성 및 관리](#ReplicationController-생성-및-관리)
    - [파드 라벨 변경](#파드-라벨-변경)
    - [ReplicationController 스펙 변경](#ReplicationController-스펙-변경)
    - [ReplicationController 삭제](#ReplicationController-삭제)
  - [4.3 ReplicaSet](#4.3-ReplicaSet)
    - [ReplicaSet과 ReplicationController 비교](#ReplicaSet과-ReplicationController-비교)
    - [ReplicaSet 생성 및 관리](#ReplicaSet-생성-및-관리)
  - [4.4 DaemonSet](#4.4-DaemonSet)
    - [특정 노드에서만 파드 실행](#특정-노드에서만-파드-실행)
  - [4.5 Job](#4.5-Job)

<!--te-->

## 4.1 Keeping Pods Healthy
파드가 노드에 예약되면 해당 노드의 Kubelet이 파드의 컨테이너를 실행시킨다. 컨테이너의 기본 프로세스가 비정상적으로 종료되면 Kubelet이 자동으로 컨테이너를 다시 시작시키기 때문에 쿠버네티스는 특별한 잡업을 하지 않아도 자체 치유 기능이 제공된다. 

하지만 어플리케이션이 프로세스 종료없이 작동을 멈추는 경우가 있다. 예를 들어, 메모리 누수가 있는 JAVA 프로그램은 OutOfMemoryErrors를 발생시키지만 JVM 프로세스는 계속 실행된다. 때문에 어플리케이션이 더 이상 제대로 작동하지 않는다는 신호를 쿠버네티스에 알리고 다시 시작하도록 하는 방법이 필요하다.

비정상적으로 종료된 컨테이너는 자동으로 다시 실행되기 때문에 어플리케이션에서 오류가 발생하면 프로세스를 종료시키는 방법을 사용할 수 있지만 모든 문제를 해결해 주진 않는다. 예를 들어 무한 루프나 교착 상태인 경우 어플리케이션이 다시 시작되도록 하려면 외부의 도움이 필요하게 된다.

### 활성 프로브
쿠버네티스는 Liveness Probes(이하 활성 프로브)를 통해 컨테이너의 상태를 확인할 수 있다. 파드 사양에서 각 컨테이너에 대해 활성 프로브를 지정하면 쿠버네티스는 주기적으로 프로브를 실행하고 프로브가 실패하면 컨테이너를 다시 시작시킨다. 활성 프로브 종류는 3가지가 있다.

- HTTP GET Probe
    - 사용자가 지정한 포트와 URL 경로로 GET 요청을 보내고 컨테이너가 응답을 하지 않거나 오류 응답 코드를 받으면 프로브는 실패로 간주하고 컨테이너를 다시 시작시킨다.
- TCP Socket Probe
    - 사용자가 지정한 포트에 TCP 연결을 수행하고 연결이 실패하면 실패고 간주하고 컨테이너를 다시 시작시킨다.
- Exec Probe
    - 컨테이너 안에 임의의 커맨드를 실행시키고 종료코드가 0이 아니면 실패로 간주하고 컨테이너를 다시 시작시킨다.

### HTTP GET Probe
`example-service/unhealthy`에 처음 5개의 요청은 정상적으로 처리하고 다음부터는 매번 오류코드를 반환하는 Node.js 앱을 작성했다. 이 앱을 기반으로 활성 프로브를 포함한 파드를 생성해보자.

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: liveness
spec:
  containers:
  - image: kbzjung359/kia-unhealthy
    name: unhealthy
    livenessProbe:
      httpGet:
        path: /
        port: 8080
```

이제 시간을 좀 기다렸다가 파드의 상태를 출력해보자.

```
$ kubectl get pods
NAME       READY   STATUS    RESTARTS   AGE
liveness   1/1     Running   1          2m33s

# 재시작되기 이전의 로그를 출력
$ kubectl logs liveness --previous

# 충분히 시간이 지나면 CrashLoopBackOff 상태가 되고 재시작하지 않는다
$ kubectl get pods
NAME       READY   STATUS             RESTARTS   AGE
liveness   0/1     CrashLoopBackOff   7          18m
```

restart 부분을 보면 이미 1번 재시작되었다. `kubectl describe`를 통해 이전의 컨테이너가 어떻게 종료되었는지 알 수 있다.

```
$ kubectl describe pods liveness
...
Containers:
  unhealthy:
    ...
    Last State: Terminated
      Reason: Error
      Exit Code: 137
      ...
    ...
    Restart Count: 1
    Liveness: http-get http://:8080/ delay=0s timeout=1s period=10s #success=1 #failure=3
    ...
Events:
  Type     Reason     Age                  From               Message
  ----     ------     ----                 ----               -------
  Warning  Unhealthy  71s (x6 over 3m21s)  kubelet            Liveness probe failed: HTTP probe failed with statuscode: 500
  Normal   Killing    71s (x2 over 3m1s)   kubelet            Container unhealthy failed liveness probe, will be restarted
...
```

이전의 컨테이너는 오류로 종료되었고 종료 코드는 137이었다. 137은 128(외부 시그널에 의해 종료), 9(SIGKILL 시그널 번호)의 합계이다. 즉 이전의 컨테이너는 SIGKILL에 의해 종료되었다는 것을 알 수 있다. 하단에 Events는 컨테이너가 종료된 이유를 볼 수 있다. 71초에 활성 프로브에 의해 컨테이너가 비정상적이라고 판단하고 컨테이너를 재시작 시켰다. 

컨테이너가 종료되면 완전히 새로운 컨테이너가 생성된다. 동일한 컨테이너에서 프로세스를 다시 시작시키는 것이 아니다. 

### 활성 프로브의 추가 변수
`kubectl describe`로 파드 정보를 출력해보면 다음과 같은 정보가 출력된다.

```
Liveness: http-get http://:8080/ delay=0s timeout=1s period=10s #success=1 #failure=3
```

- delay: 컨테이너가 시작되고 설정된 시간 이후에 프로빙이 시작
- timeout: 컨테이너는 설정된 이간 이내에 응답을 반환해야 함
- period: 설정된 시간마다 프로브됨
- failure: 설정된 횟수만큼 연속으로 실패하면 컨테이너를 다시 시작시킴

이전에 실행 시킨 파드의 프로브는 컨테이너가 시작된 직후부터 프로빙을 시작하고 1초이내에 응답을 받지 않으면 실패로 판단한다. 프로브는 10초마다 수행되고 연속으로 3번 실패하면 컨테이너를 다시 시작한다. 이러한 추가 변수는 프로브를 정의할 때 사용자가 정의할 수 있다.

```yaml
# ...
livenessProbe:
  # ...
  initialDelaySeconds: 15
```

프로브의 초기 지연시간을 설정하지 않으면 컨테이너의 앱이 시작되지 않은 상태에서 프로빙이 시작되기 때문에 정상적인 경우에도 프로브 실패로 이어질 수 있다. 

### 좋은 활성 프로브
프로덕션에서 실행되는 파드의 경우 활성 프로브를 항상 정의해야 한다. 프로브가 없으면 컨테이너의 앱이 정상적인 상태인지 확인할 수 없고 프로세스가 실행되는 한 컨테이너가 정상이라고 간주된다.

더 나은 헬스체크를 위해 `/health` 경로로 요청을 수행하도록 프로브를 설정하고 어플리케이션에서는 실행중인 중요 구성 요소에 대해 내부 상태를 확인하고 결과를 응답하는 방식을 사용하는 것이 좋다. 

이때 해당 엔드포인트에 인증이 설정되어있지 않은지 확인해야 한다. 또한 내부 상태를 확인할 때 어플리케이션 외부 요인의 영향을 받지 않는지 확인해야 한다. 예를 들어서 데이터베이스에 연결할 수 없을 때 오류를 반환하도록 하면 안된다. 실제로 근본적인 원인이 데이터베이스에 있는 경우에 컨테이너를 다시 시작해도 오류는 해결되지 않기 때문에 활성 프로브가 계속 실패하게 된다. 마지막윽로 활성 프로브에 설정한 엔드포인트는 너무 많은 리소스를 샤용하지 않고 오래걸리지 않도록 구성해야 한다. 활성 프로브는 비교적으로 자주 실행되기 때문에 컨테이너 속도가 느려질 수 있다.   

컨테이너의 프로세스가 종료되거나 활성 프로브가 실패해서 컨테이너를 다시 시작시키는 작업은 파드가 예약된 노드의 Kubelet에 의해 수행된다. 워커 노드 자체가 망가지는 경우 함께 작동 중지된 파드를 다른 노드에서 재시작 시키는 것은 Kubernetes Control Plane의 역할이지만 사용자가 직접 생성한 파드에 대해서는 해당 작업을 수행하지 않는다. 따라서 사용자가 직접 생성한 파드는 파드가 예약된 노드가 망가지면 아무것도 할 수 없다. 파드가 다른 노드에서 다시 시작되도록 하려면 파드를 ReplicationController, ReplicaSet, DaemonSet, Job 등의 메커니즘에 의해 관리되도록 해야한다.

## 4.2 ReplicationController
ReplicationController(이하 RC)는 파드가 계속 실행되도록하는 쿠버네티스 리소스이다. 노드의 장애같은 이벤트로 파드가 중지되는 경우 RC는 누락된 파드를 감지하고 교체 파드를 생성한다.

![image](https://user-images.githubusercontent.com/44857109/105858228-6ab5e600-602e-11eb-911e-5c5889fe4ece.png)

위 그림은 Node1이 다운된 이후 파드A, B에 어떤 일이 일어나는지 나타나있다. 파드A는 직접 생성되었고 파드B는 RC에 의해 관리되는 파드이다. Node1이 장애로 인해 작동이 멈추면 RC는 파드B가 중지되었다는 것을 감지하고 다른 노드에 새로운 파드를 생성하지만 파드A는 그대로 중지된다.

<br>

RC는 실행중인 파드 목록을 지속적으로 모니터링하고 설정된 파드의 수가 실제로 실행되고 있는 파드의 수와 일치하는지 확인한다. 실행중인 파드가 적으면 새로운 파드 복제본을 생성하고 많이 실행되고 있으면 초과된 복제본을 제거한다. RC가 원하는 파드의 복제본 개수를 확인하는 것은 라벨 셀렉터를 기반으로 작동된다. 따라서 RC는 다음과 같은 구성 요소가 있다.

- 파드 복제본을 생성하기 위한 파드 템플릿
- 유지하고자 하는 파드 복제본 수
- RC 범위에 있는 파드를 확인하기 위한 라벨 셀렉터

![image](https://user-images.githubusercontent.com/44857109/105860456-ef096880-6030-11eb-99b0-9a11188b15d6.png)

3개 구성 요소 모두 언제든지 수정할 수 있지만 라벨 셀렉터와 파드 템플릿의 변경은 기존에 생성된 파드에 반영되지 않는다. 라벨 셀렉터를 변경하면 기존 파드가 RC의 범위를 벗어날 수 있다. 또한 RC는 파드를 생성한 뒤 파드의 실제 내용(컨테이너 이미지, 환경 변수 등)에 대해서는 고려하지 않는다. 따라서 파드 템플릿의 변경은 이후에 생성된 파드에만 영향을 준다.

ReplicationController는 다음과 같은 기능을 제공한다.

- 기존 파드가 누락된 경우 새로운 파드를 시작해서 원하는 수의 파드가 항상 실행되고 있는지 확인한다.
- 워커 노드에 장애가 발생하면 해당 노드에서 실행중이던 모든 파드에 대한 대체 복제본을 생성한다.
  - 파드 인스턴스가 다른 노드로 재배치되는 것이 아니라 완전히 새로운 인스턴스를 생성한다.
- 수동 및 자동으로 쉽게 파드를 수평 확장할 수 있다.

### ReplicationController 생성 및 관리
YAML 디스크립터를 이용해서 RC를 생성해보자.

```yaml
apiVersion: v1
kind: ReplicationController
metadata:
  name: hostname-rc
spec:
  replicas: 2 # 복제본 수 설정
  selector: # RC의 관리 범위 지정
    app: hostname
  template: # 생성할 파드를 설정
    metadata:
      labels: 
        app: hostname # 라벨 지정
    spec:
      containers:
      - name: hostname
        image: kbzjung359/kia-hostname
        ports:
        - containerPort: 8080
```

RC 디스크립터를 작성할 때 주의해야할 점은 셀렉터의 라벨과 템플릿의 라벨이 일치하도록 해야하는 것이다. 라벨이 일치하지 않으면 RC는 파드를 실행시키고 있어도 셀렉터에 의해 추적되지 않아 새로운 파드를 계속 생성하게 된다. 이 시나리오를 방지하기 위해서 Kubernetes API는 디스크립터에서 잘못된 정의가 있는 경우 요청을 처리하지 않는다. 그래서 권장되는 방식은 RC의 라벨 셀렉터는 지정하지 않는 것이다. RC의 셀렉터가 정의되지 않으면 자동으로 템플릿의 라벨을 추출해서 지정되기 때문에 디스크립터를 더 간단하게 작성할 수 있고 실수를 예방할 수 있다.

이제 RC를 통해 파드를 생성하고 파드가 어떻게 유지되는지 확인해보자.

```
$ kubectl create -f hostname-rc.yaml
replicationcontroller/hostname-rc created

$ kubectl get pods -L app
NAME                READY   STATUS    RESTARTS   AGE   APP
hostname-rc-6j5nt   1/1     Running   0          15s   hostname
hostname-rc-7wnkx   1/1     Running   0          15s   hostname

# 파드를 수동으로 삭제
$ kubectl delete pods hostname-rc-7wnkx
pod "hostname-rc-7wnkx" deleted

# 파드 상태 확인
$ kubectl get pods -L app
NAME                READY   STATUS            RESTARTS   AGE     APP
hostname-rc-6j5nt   1/1     Running            0          7m15s   hostname
hostname-rc-7wnkx   1/1     Terminating        0          7m15s   hostname
hostname-rc-l5fdq   1/1     ContainerCreating  0          8s      hostname

# RC 상태 확인
# replicationcontroller -> rc
$ kubectl get rc
NAME          DESIRED   CURRENT   READY   AGE
hostname-rc   2         2         1       7m20s
```

파드를 수동으로 삭제하면 RC는 자동으로 새로운 파드 `hostname-rc-l5fdq`를 실행시키는 것을 볼 수 있다. 현재 복제본 수에 포함되지 않지만 종료중인 파드는 실행중인 상태로 간주되기 때문에 3개의 파드가 출력된다.

<그림>

Kubernetes API는 클라이언트가 리소스 및 리소스 목록에 대한 변경을 감시할 수 있도록 허용하기 때문에 RC는 삭제되는 파드에 대해 즉시 알림을 받는다. 하지만 이것이 새로운 복제본 파드를 생성하는 원인은 아니고 RC가 라벨 셀렉터를 통해 실제 파드 수를 확인한 뒤에 적절한 조치를 취하도록 트리거된다.

minikube를 통해 실습하고 있기 때문에 노드 장애에 따른 상태 변화 확인은 생략했다.

### 파드 라벨 변경
RC에 의해 생성되는 파드는 실제로 RC에 연결되지 않는다. 따라서 파드의 라벨을 수정해서 RC의 범위에서 제거되거나 추가되게 할 수 있다. RC의 라벨 셀렉터와 일치하지 않도록 파드의 라벨을 수정하면 파드는 수동으로 생성된 파드와 동일해진다. RC에 의해 관리되지 않기 때문에 노드에 장애가 생기면 다른 노드에서 다시 시작되지 않는다.

파드의 라벨을 추가, 수정해보고 상태 변화를 확인해보자.

```
# 새로운 라벨 추가
$ kubectl label pods hostname-rc-6j5nt type=error
pod/hostname-rc-6j5nt labeled

$ kubectl get pods --show-labels
NAME                READY   STATUS    RESTARTS   AGE   LABELS
hostname-rc-6j5nt   1/1     Running   0          27m   app=hostname,type=error
hostname-rc-l5fdq   1/1     Running   0          20m   app=hostname

# 셀렉터 범위에 있는 라벨 수정
$ kubectl label pods hostname-rc-6j5nt app=deprec --overwrite
pod/hostname-rc-6j5nt labeled

$ kubectl get pods --show-labels
NAME                READY   STATUS              RESTARTS   AGE   LABELS
hostname-rc-6j5nt   1/1     Running             0          29m   app=deprec,type=error
hostname-rc-l5fdq   1/1     Running             0          22m   app=hostname
hostname-rc-vtxtl   0/1     ContainerCreating   0          2s    app=hostname
```

RC는 라벨 셀렉터에 지정된 라벨이 있는지 여부만 검사하기 때문에 파드에 새로운 라벨이 추가돼도 여전히 RC의 범위에 있다고 간주한다. 때문에 새로운 파드 복제본이 생성되지 않았다. 반면에 RC 라벨 셀렉터 범위에 있는 app 라벨을 수정한 경우 `hostname-rc-6j5nt`는 RC의 범위를 벗어나게 되고 RC는 복제본 수를 유지하기 위해 새로운 파드를 생성한다. 

<그림>

보통 파드를 RC 범위 밖으로 이동시키는 작업은 오류가 생긴 파드가 생겼을 때 파드를 RC 범위 밖으로 옮기고 새로운 파드가 생성되도록 한 다음 파드를 원하는 방식으로 디버기하는 식으로 작업할 때 사용한다.

만약 RC의 라벨 셀렉터를 수정한다면 이전에 생성된 파드는 모두 RC의 범위를 벗어나게되고 RC는 복제본 수에 맞게 새로운 파드를 생성하게 된다. 

### ReplicationController 스펙 변경
RC의 파드 템플릿은 언제든지 수정할 수 있다. 수정된 파드 템플릿은 이후에 생성된 파드에만 영향을 미치고 이전에 생성된 파드에는 영향을 주지 않는다. 이전 파드를 수정하려면 해당 파드를 삭제하고 RC가 새로운 템플릿을 기반으로 새 파드를 생성하도록 해야한다.

<그림>

RC를 편집해서 파드 템플릿의 라벨을 추가해보자. `kubectl edit`을 통해 수정할 수 있다.

```
$ kubectl edit rc hostname-rc
...
template:
  metadata:
    labels:
      type: hotfix #새로운 라벨 추가
...
replicationcontroller/hostname-rc edited
```

이제 기존의 파드 한개를 삭제하면 새로 수정된 파드 템플릿을 기반으로 새 파드가 생성된다.

```
$ kubectl delete pods hostname-rc-l5fdq

$ kubectl get pods --show-labels
NAME                READY   STATUS              RESTARTS   AGE   LABELS
hostname-rc-8rzgh   0/1     ContainerCreating   0          2s    app=hostname,type=hotfix
hostname-rc-l5fdq   1/1     Terminating         0          44m   app=hostname
hostname-rc-vtxtl   1/1     Running             0          22m   app=hostname
```

RC가 관리할 파드의 복제본 수를 변경하는 것도 간단하다. 파드 수를 늘리고 줄이는 것은 ReplicationController 리소스에서 replicas 필드 값을 변경하는 것으로 모든 작업이 수행된다. `kubectl edit`을 통해 리소스를 직접 수정하거나 다음 명령으로 수정할 수 있다.

```
$ kubectl scale rc hostname-rc --replicas=3
replicationcontroller/hostname-rc scaled

$ kubectl get pods --show-labels
hostname-rc-8rzgh   1/1     Running             0          4m29s   app=hostname,type=hotfix
hostname-rc-lqkmb   0/1     ContainerCreating   0          2s      app=hostname,type=hotfix
hostname-rc-vtxtl   1/1     Running             0          26m     app=hostname
```

쿠버네티스에서 파드를 수평적으로 확장하는 것은 "인스턴스를 몇 개만큼 실행하고 싶습니다."라고 선언하기만 하면 된다. 쿠버네티스에게 명시적으로 무엇을 어떻게 수행해야하는지 지정하는 것이 아니기 때문에 작업량이 적고 간편하게 쿠버네티스와 상호작용할 수 있다.

### ReplicationController 삭제
`kubectl delete`를 이용해서 RC를 삭제하면 RC가 관리하고 있던 파드도 함께 삭제된다. 하지만 `--cascade` 옵션을 이용해서 RC만 삭제하고 파드는 실행 상태로 둘 수 있다. 이 방법은 파드는 그대로 두고 파드를 관리하는 컨트롤러를 교체하는 작업에 유용하다. 

```
$ kubectl delete rc hostname-rc --cascade=false
```

## 4.3 ReplicaSet
초창기에 쿠버네티스에서 파드를 복제하고 재시작시키는 리소스 유형은 ReplicationController 밖에 없었다. 나중에 RC와 유사한 컨트롤러인 ReplicaSet(이하 RS)가 도입되었고 RC를 완전히 대체했다. 일반적으로 컨트롤러는 직접적으로 생성하지 않고 상위 레벨 리소스에 의해 자동으로 생성된 것을 사용할 것이지만 어차피 개념을 알고 넘어가야 한다.

### ReplicaSet과 ReplicationController 비교
RS는 RC와 똑같이 작동하지만 파드의 범위를 지정하는 셀렉터의 표현력이 더 뛰어나다. RC의 라벨 셀렉터는 지정된 라벨과 일치하는 파드만 허용하지만 RS는 특정 라벨이 없거나 라벨의 키만 일치하는 파드도 허용할 수 있다. RS는 라벨이 각각 `env=prod`, `env=devel`인 파드들을 하나의 그룹으로 관리할 수 있다. 또한 키가 `env`인 라벨을 갖고있는 모든 파드들도 한 그룹으로 관리할 수 있다.


### ReplicaSet 생성 및 관리
이전에 작성했던 ReplicationController를 ReplicaSet으로 대체하는 YAML 디스크립터를 작성해보자.

```yaml
apiVersion: apps/v1beta2 # ReplicaSet은 v1beta2 API 그룹에 있다.
kind: ReplicaSet
metadata:
  name: hostname-rs
spec:
  replicas: 2
  selector:
    matchLabels: # ReplicaSet에서 보완된 셀렉터 유형
      app: hostname
  template:
    metadata:
      labels:
        app: hostname
    spec:
      containers:
      - name: hostname
        image: kbzjung359/kia-hostname
```

RC와 비교해서 apiVersion 그룹이 v1beta2로 변경되었고 셀렉터 속성 바로 아래에 라벨을 지정하는 대신 matchLabels 속성이 추가되었다. 이외에 파드 템플릿은 RC와 동일하다.

> apiVersion은 API 그룹, 실제 API 버전으로 나뉜다. 쿠버네티스의 코어 API 그룹에 있는 리소스들은 API 그룹을 지정할 필요가 없다. Pod 리소스를 지정할 때 `v1`으로 API 그룹을 명시하지 않았다. 이후 쿠버네티스 버전에서 추가된 리소스들은 세부적인 API 그룹으로 나뉘어져 있기 때문에 API 그룹을 명시해야 한다. ReplicaSet의 경우 API 그룹은 `apps`이다. 이후에 실제 버전인 `v1beta2`을 표기한다.

<br>

RC에 비해 RS의 주요 개선 사항은 라벨 셀렉터이다. RS는 RC의 셀렉터와 동일하게 작동하는 `matchLabels`과 더불어 `matchExpressions`을 통해 더 강력한 셀렉터를 지원한다.

```yaml
selector:
  matchExpressions:
  - key: app
    operator: In
    values:
    - hostname
```

셀렉터에는 여러개의 표현식을 추가할 수 있다. 이때 각 표현식에는 `key`, `operator`는 필수로 포함되어야 한다. 연산자는 다음 4가지가 있다.

- In: `values` 속성에 지정된 값중에 하나가 라벨의 값과 일치해야 함.
- NotIn: `values` 속성에 지정된 값중에 하나라도 라벨의 값과 매치되지 않아야 함.
- Exists: 갖고 있는 라벨의 키중에 지정된 키가 포함되어야 함. 이때 `values` 속성은 지정하지 않는다. 
- DoesNotExist: 갖고 있는 라벨의 키중에 지정된 키가 포함되지 않아야 함. 이때 `values` 속성은 지정하지 않는다.

셀렉터에 여러개의 표현식을 지정하는 경우 셀렉터가 파드와 일치하려면 모든 표현식이 true로 평가되어야 한다.

<br>

ReplicationController와 마찬가지로 ReplicaSet를 삭제하면 셀렉터에 일치하는 모든 파드가 함께 삭제된다. 

```
# replicaset -> rs
$ kubectl delete rs hostname-rs
```

## 4.4 DaemonSet
ReplicationController와 ReplicaSet는 모두 쿠버네티스 클러스터의 임의의 노드들에서 특정 수의 파드를 실행하는데 사용된다. 이와 반대로 클러스터의 모든 노드에서 파드를 실행해야하는 경우가 있다.

![image](https://user-images.githubusercontent.com/44857109/105988412-ebceb500-60e2-11eb-8aa2-0e4f83214cdc.png)

일반적으로 시스템 수준의 작업을 수행하는 인프라 관련 파드를 실행하는 경우에 사용한다. 예를 들어 쿠버네티스의 `kube-proxy`는 클러스터 구성을 위해 모든 노드에서 실행되어야 한다. 이러한 구성이 필요한 경우에 DaemonSet(이하 DS)을 사용한다. 

DS는 클러스터를 구성하는 노드 수만큼 파드를 생성하고 파드를 각 노드에 배포한다. DS는 복제본 수에 대한 개념이 없다. 작업은 파드 셀렉터와 일치하는 파드가 가 노드에서 실행되고 있는지 확인하는 것이기 때문에 복제를 고려하지 않는다. 노드가 다운되면 DS로 인해 파드가 다른 곳에 생성되지 않는다. 하지만 새로운 노드가 클러스터에 추가되면 DS는 즉시 새 파드 인스턴스를 해당 노드에 배포한다. 누군가가 실수로 DS 범위에 있는 파드를 삭제하는 경우에 DS는 RS와 마찬가지로 구성된 파드 템플릿에 따라 해당노드에 새로운 파드를 생성한다.

### 특정 노드에서만 파드 실행
DaemonSet은 클러스터의 모든 노드에 파드를 배포하지만 노드 셀렉터를 이용해서 특정 노드 그룹에서만 실행되도록 지정할 수도 있다. 

> 쿠버네티스에는 특정 노드에 예약할 수 없게 만들어서 파드가 해당 노드에 배포되지 않도록 할 수 있다. 하지만 DaemonSet에 의해 관리되는 파드는 기본적으로 스케줄러를 거치지 않고 배포되기 때문에 이러한 노드에도 예약될 수 있다. DaemonSet은 일반적으로 시스템 서비스를 실행하기 위한 것이기 때문에 바람직한 작동이다.

SSD가 있는 모든 노드에서 실행해야 하는 데몬을 실행한다고 가정해보자. 해당 노드들은 클러스터 관리자에 의해 `disk=ssd` 라벨이 추가되어 있기 때문에 해당 라벨을 선택하는 노드 셀렉터로 DS를 만들면 된다.

![image](https://user-images.githubusercontent.com/44857109/105992188-22f39500-60e8-11eb-8021-3f7c39af4810.png)

5초마다 STDOUT에 "SDD OK"를 출력하는 프로세스를 DS로 만드는 디스크립터를 만들어보자. 이 도커 이미지는 직접 빌드하지 않고 책에서 주는 이미지를 사용했다.

```yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: ssd-monitor
spec:
  selector:
    matchLabels:
      app: ssd-monitor
    template:
      metadata:
        labels:
          app: ssd-monitor 
      spec:
        nodeSelector:
          disk: ssd
        containers:
        - name: main
          image: luksa/ssd-monitor
```

이제 노드에 `disk=ssd` 라벨을 추가하고 DS를 생성해보자.

```
$ kubectl create -f ssd-monitor.yaml

$ kubectl label node minikube disk=ssd

$ kubectl get pods
NAME                READY   STATUS              RESTARTS   AGE
ssd-monitor-fv5pp   0/1     ContainerCreating   0          3s

# daemonset -> ds
$ kubectl get ds
NAME          DESIRED   CURRENT   READY   UP-TO-DATE   AVAILABLE   NODE SELECTOR   AGE
ssd-monitor   1         1         1       1            1           disk=ssd        2m52s
```

노드에 해당 라벨을 제거하면 어떻게 될까? 당연히 실행중이던 파드가 종료된다. 이건 간단해서 실습은 넘어갔다.

## 4.5 Job
지금까지 공부한 ReplicationController, ReplicaSet, DaemonSet은 작업이 완료되고 종료되는 상황을 고려하지 않고 프로세스가 종료되면 자동으로 다시 시작시킨다. 하지만 작업이 완료되면 종료되는 작업을 실행해야 하는 경우가 있다. 이를 위해선 프로세스가 종료된 후에 다시 시작시키지 않는 컨트롤러가 필요하다. Job 리소스는 내부에서 실행중인 프로세스가 성공적으로 완료될 때 컨테이너가 다시 시작되지 않는 파드를 실행할 수 있다.

Job이 실행되고 있던 노드가 장애로 다운되면 ReplicaSet와 같은 방식으로 다른 노드에 다시 예약된다. 프로세스 자체가 실패한 경우(프로세스의 종료코드가 오류 종료 코드인 경우) 컨테이너를 다시 시작하거나 다시 시작하지 않도록 작업을 구성할 수 있다. 아래 그림은 Job과 ReplicaSet에 의해 생성된 파드의 생명 주기를 보여준다.

![image](https://user-images.githubusercontent.com/44857109/105997515-0c9d0780-60ef-11eb-981a-90e69a729bb9.png)

Job은 작업이 정상적으로 완료되는 것이 중요한 임시 작업에 유용하다. 어떠한 컨트롤러에도 관리되지 않는 파드를 생성하고 완료될 때까지 기다리는 방법도 있지만, 이 방법은 노드가 다운되는 경우 대처 방안이 없다. 보통 어떤 위치에 저장되어 있는 데이터를 가공해서 다른 곳으로 내보내야 하는 배치작업에 사용된다.

### Job 생성 및 관리
2분동안 Sleep하고 종료되는 컨테이너를 기반으로 Job을 실행해보자.

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: batch-job
spec:
  template:
    metadata:
      labels:
        app: batch-job
    spec:
      restartPolicy: OnFailure # 비정상 종료일 때 재시작
      containers:
      - name: main
        image: luksa/batch-job
```

`restartPolicy` 속성을 통해서 실행중인 프로세스가 종료될 때 쿠버네티스가 수행해야 하는 작업을 지정할 수 있다. 이 속성은 기본값이 Always이지만 Job은 무기한 실행되지 않는 작업이기 때문에 항상 OnFailure 또는 Never 옵션을 명시적으로 지정해야 한다.

이제 Job을 실행해보자.

```
$ kubectl create -f batch-job.yaml

$ kubectl get job
NAME        COMPLETIONS   DURATION   AGE
batch-job   0/1           31s        31s

$ kubectl get pods
NAME              READY   STATUS    RESTARTS   AGE
batch-job-9zkdc   1/1     Running   0          19s


# 2분 뒤
$ kubectl get job
NAME        COMPLETIONS   DURATION   AGE
batch-job   1/1           2m9s       2m11s

$ kubectl get pods
NAME              READY   STATUS      RESTARTS   AGE
batch-job-9zkdc   0/1     Completed   0          2m19s
```

## 여러 파드 인스턴스 실행
Job은 둘 이상의 파드 인스턴스를 생성하고 병렬 또는 순차적으로 실행하도록 구성할 수 있다. `completions`, `parallelism` 속성을 통해 설정할 수 있다.

```yaml
# ...
spec:
  completions: 5
  template:
    # ...
# ...
```

위의 디스크립터로 생성된 Job은 파드를 순차적으로 5개의 파드를 실행시킨다. 중간에 오류로 인해 실패하는 파드가 있을 수 있기 때문에 5개 이상의 파드가 생성될 수 있다.

`paralleism` 속성을 통해 하나의 파드를 차례로 실행하는 대신 여러 파드를 병렬로 실행하도록 할 수 있다.

```yaml
# ...
spec:
  comletions: 5
  paralleism: 2
  template:
    # ...
# ...
```

위의 디스크립터로 생성된 Job은 최대 2개의 파드를 병렬로 실행하고, 그중 하나의 파드가 성공적으로 종료되면 이후 5개의 파드가 성공적으로 종료될 때까지 다음 파드를 실행한다.

병렬적으로 Job이 실행되는 도중에 병렬로 실행되는 파드의 수를 변경할 수 있다. 이 작업은 ReplicaSet과 유사하게 `kubectl scale`을 통해 수행한다.

```
$ kubectl scale job multi-batch-job --replicas 3
```

## 4.6 CronJob
보통 배치 작업은 미래의 특정 시간에 실행되거나 지정된 간격으로 반복적으로 실행된다. Linux 같은 운영체제에서 이러한 작업을 cron 작업이라 부른다. 쿠버네티스도 이 작업을 지원한다.

CronJob 디스크립터는 다음과 같이 작성할 수 있다.

```yaml
apiVersion: batch/v1beta1
kind: CronJob
metadata:
  name: batch-job-every-fifteen-minutes
spec:
  schedule: "0,15,30,45 * * * *" # 매일 매 시간 0, 15, 30, 45분에 실행
  jobTemplate:
    spec:
      template:
        metadata:
          labels:
            app: periodic-batch-job
        spec:
          restartPolicy: OnFailure
          containers:
          - name: main
            image: luksa/batch-job
```

> `schedule` 속성의 구성은 cron 일정 형식과 동일하다. 구성요소는 분, 시, 날짜, 달, 요일로 구성된다. 예를들어 30분마다 실행하고 매월 1일에만 실행하려면 `0,30 * 1 * *`으로 표현할 수 있다. 또한 매주 일요일 오전 3시에 실행하려면 `0 3 * * 0`으로 표현할 수 있다.

CronJob은 일정 시간마다 Job 리소스를 생성하고, 생성된 Job은 Pod를 생성한다. 이때 Job이나 Pod가 상대적으로 느리게 생성되고 실행될 수 있다. Job이 예정된 시간보다 너무 늦게 시작되지 않도록 하기 위한 요구사항이 있을 수 있다. 이 경우 `startingDeadlineSeconds` 속성을 지정해서 특정 시간 안에 실행되지 않으면 Job이 실행되지 않게 설정할 수 있다.

```yaml
# ...
spec:
  schedule: "0,15,30,45 * * * *"
  startingDeadlineSeconds: 15
  # ...
# ...
```

이 경우 10:30:00에 Job이 시작되어야하지만 10:30:15까지 실행되지 않으면 Job은 실행되지 않고 실패로 표시된다.

정상적으로 CronJob이 실행되는 경우 지정된 일정에 따라 항상 하나의 Job만 생성하지만, 두 개의 Job이 동시에 실행되거나 아예 생성되지 않는 경우가 생길 수 있다. 이 문제에 대처하기 위해서 Job은 여러번 실행해도 오류가 나지 않는 작업(멱등성)이어야 하고 이전의 Job에서 누락된 작업이 다음 Job에서 처리될 수 있도록 구성해야 한다.
