# Replication and Other Controller

## Keeping Pods Healthy
파드가 노드에 예약되면 해당 노드의 Kubelet이 파드의 컨테이너를 실행시킨다. 컨테이너의 기본 프로세스가 비정상적으로 종료되면 Kubelet이 자동으로 컨테이너를 다시 시작시키기 때문에 쿠버네티스는 특별한 잡업을 하지 않아도 자체 치유 기능이 제공된다. 

하지만 어플리케이션이 프로세스 종료없이 작동을 멈추는 경우가 있다. 예를 들어, 메모리 누수가 있는 JAVA 프로그램은 OutOfMemoryErrors를 발생시키지만 JVM 프로세스는 계속 실행된다. 때문에 어플리케이션이 더 이상 제대로 작동하지 않는다는 신호를 쿠버네티스에 알리고 다시 시작하도록 하는 방법이 필요하다.

비정상적으로 종료된 컨테이너는 자동으로 다시 실행되기 때문에 어플리케이션에서 오류가 발생하면 프로세스를 종료시키는 방법을 사용할 수 있지만 모든 문제를 해결해 주진 않는다. 예를 들어 무한 루프나 교착 상태인 경우 어플리케이션이 다시 시작되도록 하려면 외부의 도움이 필요하게 된다.

### Liveness Probes
쿠버네티스는 활성 프로브(Liveness Probes)를 통해 컨테이너의 상태를 확인할 수 있다. 파드 사양에서 각 컨테이너에 대해 활성 프로브를 지정하면 쿠버네티스는 주기적으로 프로브를 실행하고 프로브가 실패하면 컨테이너를 다시 시작시킨다. 활성 프로브 종류는 3가지가 있다.

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
  inittialDelaySeconds: 15
```

프로브의 초기 지연시간을 설정하지 않으면 컨테이너의 앱이 시작되지 않은 상태에서 프로빙이 시작되기 때문에 정상적인 경우에도 프로브 실패로 이어질 수 있다. 

### 좋은 활성 프로브
프로덕션에서 실행되는 파드의 경우 활성 프로브를 항상 정의해야 한다. 프로브가 없으면 컨테이너의 앱이 정상적인 상태인지 확인할 수 없고 프로세스가 실행되는 한 컨테이너가 정상이라고 간주된다.

더 나은 헬스체크를 위해 `/health` 경로로 요청을 수행하도록 프로브를 설정하고 어플리케이션에서는 실행중인 중요 구성 요소에 대해 내부 상태를 확인하고 결과를 응답하는 방식을 사용하는 것이 좋다. 

이때 해당 엔드포인트에 인증이 설정되어있지 않은지 확인해야 한다. 또한 내부 상태를 확인할 때 어플리케이션 외부 요인의 영향을 받지 않는지 확인해야 한다. 예를 들어서 데이터베이스에 연결할 수 없을 때 오류를 반환하도록 하면 안된다. 실제로 근본적인 원인이 데이터베이스에 있는 경우에 컨테이너를 다시 시작해도 오류는 해결되지 않기 때문에 활성 프로브가 계속 실패하게 된다. 마지막윽로 활성 프로브에 설정한 엔드포인트는 너무 많은 리소스를 샤용하지 않고 오래걸리지 않도록 구성해야 한다. 활성 프로브는 비교적으로 자주 실행되기 때문에 컨테이너 속도가 느려질 수 있다.   

컨테이너의 프로세스가 종료되거나 활성 프로브가 실패해서 컨테이너를 다시 시작시키는 작업은 파드가 예약된 노드의 Kubelet에 의해 수행된다. 워커 노드 자체가 망가지는 경우 함께 작동 중지된 파드를 다른 노드에서 재시작 시키는 것은 Kubernetes Control Plane의 역할이지만 사용자가 직접 생성한 파드에 대해서는 해당 작업을 수행하지 않는다. 따라서 사용자가 직접 생성한 파드는 파드가 예약된 노드가 망가지면 아무것도 할 수 없다. 파드가 다른 노드에서 다시 시작되도록 하려면 파드를 ReplicationController, ReplicaSet, DaemonSet, Job 등의 메커니즘에 의해 관리되도록 해야한다.
