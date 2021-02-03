# Chapter 9. Deployments: updating applications declaratively

<!--ts-->
  - [9.1 Updating Applications Running in Pods](#9.1-Updating-Applications-Running-in-Pods)
    - [기존 파드를 모두 삭제](#기존-파드를-모두-삭제)
    - [새로운 파드를 한번에](#새로운-파드를-한번에)
    - [점진적으로 새로운 파드 생성](#점진적으로-새로운-파드-생성)
  - [9.2 Performing an Automatic Rolling Update with a ReplicationController](#9.2-Performing-an-Automatic-Rolling-Update-with-a-ReplicationController)
    - [rolling-update가 폐기된 이유](#rolling-update가-폐기된-이유)
  - [9.3 Using Deployments for Updating Apps Declaratively](#9.3-Using-Deployments-for-Updating-Apps-Declaratively)
    - [Deployment 생성](#Deployment-생성)
    - [Deployment 업데이트](#Deployment-업데이트)
    - [Deployment 롤백](#Deployment-롤백)
    - [롤아웃 속도 제어](#롤아웃-속도-제어)
    - [롤아웃 프로세스 일시 정지 시키기](#롤아웃-프로세스-일시-정지-시키기)
    - [잘못된 버전 출시 방지](#잘못된-버전-출시-방지)
<!--te-->

## 9.1 Updating Applications Running in Pods
실행중인 서비스를 업데이트하기 위해 v1 이미지로 실행중이던 파드를 v2 이미지를 실행하는 파드로 교체하는 상황을 생각해보자. 파드가 생성된 후에는 기존 파드의 이미지를 변경할 수 없기 때문에 이전 파드를 제거하고 새로운 파드로 교체해야 한다. 파드를 교체하는 방법은 2가지가 있다.

- 먼저 기존 파드를 모두 삭제하고 다음 새 파드를 실행한다.
- 새로운 파드를 생성하고 해당 파드가 실행되면 이전의 파드를 제거한다. 
    - 새로운 파드를 모두 추가한 뒤에 이전 파드를 한번에 제거한다.
    - 점진적으로 새로운 파드를 추가하고 이전 파드를 제거해서 순차적으로 진행한다.

두 방식 모두 장단점이 있다. 전자의 방식은 업데이트를 진행하는 동안 서비스를 중단해야 한다. 후자의 방식은 서비스를 중단하지 않아도 되지만 새로운 버전이 이전 버전과 동시에 실행되어도 충돌이 발생하지 않도록 해야한다.

먼저 수동으로 파드를 업데이트해보자.

### 기존 파드를 모두 삭제
ReplicationController의 파드 템플릿은 언제든지 수정할 수 있다. RC의 파드 템플릿에서 새로운 버전의 이미지를 실행하도록 수정하고 이전에 생성된 모든 파드를 삭제하면 새로운 템플릿으로 파드가 생성될 것이다.

이전 파드가 삭제되고 새로운 파드가 시작되는 시간 사이에 짧은 다운 타임을 허용할 수 있는 경우 제일 간단한 업데이트 방법이다.

![image](https://user-images.githubusercontent.com/44857109/106551380-75a1d680-6558-11eb-9e07-a314f7712ab6.png)

### 새로운 파드를 한번에 
다운 타임을 허용할 수 없고 이전 버전과 함께 실행될 수 있는 이미지라면 먼저 새로운 파드를 생성한 뒤에 다음 파드를 제거한다. 잠시동안 동시에 실행되는 파드의 수가 증가하기 때문에 더 많은 하드웨어 리소스가 필요하다.

새로운 버전의 파드를 시작하는 동안 서비스는 이전 버전만 가리키도록 구성한다. 이후 모든 새 파드가 시작되면 서비스의 라벨 셀렉터를 변경해서 새로운 파드를 가리키도록 한다. 이후 새 버전이 오류없이 잘 작동하는 것을 확인한 후에는 이전 버전의 컨트롤러를 삭제한다. 이 방식을 `blue green` 배포라 한다.

![image](https://user-images.githubusercontent.com/44857109/106551863-7e46dc80-6559-11eb-8418-c92c2221eab2.png)

### 점진적으로 새로운 파드 생성
모든 새 파드를 실행시키고 이전 파드를 한번에 삭제하는 대신 점진적으로 추가, 삭제하는 방식을 `rolling update` 배포라 한다. 단계적으로 이전 버전의 컨트롤러를 축소하고 새 버전의 컨트롤러를 확장하면 된다. 서비스는 두 버전의 파드를 모두 가리키도록 구성해서 클라이언트 요청을 두 파드 세트에 전달시키도록 한다. 

![image](https://user-images.githubusercontent.com/44857109/106551881-869f1780-6559-11eb-90eb-42afebd5b338.png)

이 방식을 수동으로 진행하는 것은 힘들고 오류가 발생하기 쉽다. 쿠버네티스는 단일 명령으로 이 방식의 업데이트를 수행할 수 있다.

<br>

## 9.2 Performing an Automatic Rolling Update with a ReplicationController
`kubectl rolling-update`를 사용하면 ReplicationController를 쉽게 업데이트할 수 있다. 이 방법은 오래된 방법이지만 쿠버네티스 초기에 롤링 업데이트를 수행하는 첫 번째 방법이었고 너무 많은 추가 개념을 도입하지 않고 수행할 수 있기 때문에 한번 보고 넘어가자.

<br>

먼저 hostname를 반환하는 간단한 node 서버에 버전을 함께 출력하도록 수정하고 v1, v2 이미지를 만들었다. 이후 ReplicationController와 Service를 이용해서 v1 이미지를 배포해보자.

```yaml
apiVersion: v1
kind: ReplicationController
metadata:
  name: hostname-v1
spec:
  replicas: 3
  template:
    metadata:
      name: hostname
      labels:
        app: hostname
    spec:
      containers:
      - image: kbzjung359/kia-hostname:v1
        name: nodejs
---
apiVersion: v1
kind: Service
metadata:
  name: hostname
spec:
  type: NodePort
  selector:
    app: hostname
  ports:
  - port: 80
    targetPort: 8080
    nodePort: 30123
```

> YAML 디스크립터는 `---`으로 구분해서 여러 리소스 정의를 할 수 있다.

이제 `kbzjung359/kia-hostname:v1` 이미지를 `kbzjung359/kia-hostname:v2`로 업데이트 해보자. `kubectl rolling-update` 명령어에 전달해줘야 하는 정보는 교체할 컨트롤러 이름, 새로운 컨트롤러 이름, 교체할 새 이미지이다.

```
$ kubectl get rc
NAME          DESIRED   CURRENT   READY   AGE
hostname-v1   3         3         3       5m29s

$ kubectl get po
NAME                READY   STATUS    RESTARTS   AGE
hostname-v1-bn5sq   1/1     Running   0          5m37s
hostname-v1-nqcm8   1/1     Running   0          5m37s
hostname-v1-s8jlz   1/1     Running   0          5m37s

$ kubectl rolling-update hostname-v1 hostname-v2 --image=kbzjung359/kia-hostname:v2
```

> 진행하다가 문제가 생겼다. 내가 설치한 쿠버네티스 버전 `v1.20.0`에서 `kubectl rolling-update`를 수행할 수 없는 것 같다. `kubectl --help` 해도 해당 옵션이 안나온다.. 크흠. 실습은 못하고 정리만 하자.

아무튼 해당 명령어를 실행하면 hostname-v2로 새로운 컨트롤러가 생성된다. 이때 replicas는 0으로 초기화되어있기 때문에 새로운 파드가 바로 생성되진 않는다.

새로 생성된 컨트롤러의 라벨 셀렉터를 살펴보면 app이외에도 RC를 위한 deployment 라벨이 추가되어 있다. 새로운 컨트롤러는 이전 컨트롤러가 관리하던 파드와 충돌하지 않아야 하기 때문에 추가된 것이지만 이럴려면 이전 컨트롤러에도 deployment 라벨을 추가해야 한다. 확인해보면 진짜 추가되어있다. 

![image](https://user-images.githubusercontent.com/44857109/106560372-46946080-656a-11eb-8a0a-f6080622c6f5.png)

이전 컨트롤러의 파드들에도 deployment 라벨을 수정하기 위해 rolling update는 이전에 실행되던 파드를 모두 죽이는 것을 알 수 있다. 이제 새로운 컨트롤러의 복제본 수를 늘리면 다음 그림과 같이 파드가 생성되고 이전 파드는 제거된다.

![image](https://user-images.githubusercontent.com/44857109/106560553-8c512900-656a-11eb-9d85-319ef41c4d36.png)


### rolling-update가 폐기된 이유
rolling update는 쿠버네티스가 컨트롤러의 라벨 셀렉터를 수정한다. 이 것은 사용자가 예측하기 힘들고 나중에 라벨이 충돌해서 문제가 생길 가능성이 있다. 

더 중요한 문제는 라벨 셀렉터를 수정하고 새로운 컨트롤러를 생성하고 파드를 생성, 삭제하는 모든 절차가 kubectl 클라이언트에 의해 실행된다는 것이다. 일련의 작업이 서버에서 실행되지 않기 때문에, 만약 업데이트를 진행하는 도중에 네트워크 연결이 끊어진다면 리소스들이 업데이트 과정 중간 상태에 머무르게 될 수 있다. 

## 9.3 Using Deployments for Updating Apps Declaratively
Deployment는 선언적으로 어플리케이션은 배포하고 업데이트하기 위한 ReplicationController, ReplicaSet의 상위 리소스이다. Deployment를 생성하면 ReplicaSet 리소스가 생성된다. 따라서 실제 파드는 ReplicaSet에 의해 관리된다.

![image](https://user-images.githubusercontent.com/44857109/106561755-95db9080-656c-11eb-97f2-ee654d7a654b.png)

### Deployment 생성
Deployment를 생성하는 것은 ReplicationController와 다르지 않다. RC와 마찬가지로 라벨 셀렉터, 복제본 개수, 파드 템플릿으로 구성되고 리소스가 수정될 때 업데이트를 수행하는 방법을 정의하는 필드가 추가된다. 

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: hostname
spec:
  replicas: 3
  selector:
    matchLabels:
      app: hostname
  template:
    metadata:
      name: hostname
      labels:
        app: hostname
    spec:
      containers:
      - image: kbzjung359/kia-hostname:v1
        name: nodejs
        ports:
        - containerPort: 8080
```

> Deployment는 기본적으로 여러 버전을 포함하기 때문에 이름에 특정 버전을 명시하면 안된다. apiVersion이 app/v1으로 수정되면서 라벨 셀렉터를 명시적으로 작성해야 하는 것 같다.

```
# 배포 과정을 추적할 수 있게 --record 옵션을 추가
$ kubectl create -f hostname-dp.yaml --record

# 배포 상태를 확인하는 명령어 
$ kubectl rollout status deployment hostname
deployment "hostname" successfully rolled out

$ kubectl get po --show-labels
NAME                        READY   STATUS    RESTARTS   AGE     LABELS
hostname-7cf6d977b9-fl8lh   1/1     Running   0          4m17s   app=hostname,pod-template-hash=7cf6d977b9
hostname-7cf6d977b9-w4xmj   1/1     Running   0          4m17s   app=hostname,pod-template-hash=7cf6d977b9
hostname-7cf6d977b9-wc8gl   1/1     Running   0          4m17s   app=hostname,pod-template-hash=7cf6d977b9
```

이전에 ReplicaSet으로 파드를 생성했을 때는 `hostname-fl8lh` 같이 임의로 생성된 문자열이 1개만 추가됐는데 Deployment는 2개가 추가되었다. 중간에 추가된 문자열(7cf6d977b9)은 파드 템플릿의 해시값으로 해당 파드를 관리하는 ReplicaSet의 이름으로 쓰인다.

```
$ kubectl get rs
NAME                  DESIRED   CURRENT   READY   AGE
hostname-7cf6d977b9   3         3         3       12m
```

<br>

### Deployment 업데이트
ReplicationController를 사용해서 업데이트를 수행할 때는 수행해야 하는 작업을 모두 명시적으로 지시해야 했다. 또한 rolling-update 명령은 새로운 RC 이름을 지정하고 모든 kubectl가 작업을 끝낼때까지 기다려야 했다. Deployment는 업데이트를 위해 리소스에 정의된 파드 템플릿을 수정하면 쿠버네티스가 알아서 모든 작업을 수행한다.  

업데이트 방법은 `RollingUpdate`와 `Recreate`가 있다. `Recreate`는 RC의 파드 템플릿을 수정하고 모든 파드를 삭제하는 것과 유사하게 모든 이전 파드를 삭제하고 새로운 파드를 생성하는 전략이다. `Recreate`는 새로운 파드가 생성되기 전에 모든 이전 파드를 삭제하기 때문에 어플리케이션이 여러 버전을 병렬로 실행하는 것을 지원하지 않는다. 새 버전을 시작하기 전에 이전 버전을 완전히 중지해야 하는 경우 사용하고 서비스를 완전히 사용할 수 없게되는 다운 타임이 포함된다.

`RollingUpdate`는 이전 파드를 하나씩 제거하는 동시에 새 파드를 생성한다. 전체 프로세스에서 어플리케이션을 계속 사용할 수 있도록 유지하고 요청을 처리할 수 있는 처리량이 낮아지지 않도록 한다. 이전 버전과 새 버전이 동시에 실행될 수 있는 경우 사용한다. 이 옵션이 디폴트 값이다.

<br>

이제 v1이미지를 v2로 업데이트하기 전에 진행 상황을 확인할 수 있게 업데이트 프로세스를 느리게 수행되도록 설정하자. `kubectl patch`를 사용해서 minReadySeconds를 10초로 설정한다.

> `kubectl patch`는 텍스트 변집기로 리소스 정의를 수정하지 않고 단일 속성 또는 제한된 속성을 수정할 때 유용하다.

```
$ kubectl patch deployment hostname -p '{"spec":{"minReadySeconds":10}}'
deployment.apps/hostname patched
```

이 명령어에서는 파드 템플릿을 수정하지 않았기 때문에 롤아웃이 트리거되지 않는다. 이제 이미지를 v2로 수정해보자.

```
$ kubectl set image deployment hostname nodejs=kbzjung359/kia-hostname:v2
deployment.apps/hostname image updated

$ kubectl get po
NAME                        READY   STATUS              RESTARTS   AGE
hostname-5ff7b9fdf6-dvlvm   0/1     ContainerCreating   0          8s
hostname-7cf6d977b9-fl8lh   1/1     Running             0          53m
hostname-7cf6d977b9-w4xmj   1/1     Running             0          53m
hostname-7cf6d977b9-wc8gl   1/1     Running             0          53m

$ kubectl get po
NAME                        READY   STATUS    RESTARTS   AGE
hostname-5ff7b9fdf6-dvlvm   1/1     Running   0          17s
hostname-7cf6d977b9-fl8lh   1/1     Running   0          53m
hostname-7cf6d977b9-w4xmj   1/1     Running   0          53m
hostname-7cf6d977b9-wc8gl   1/1     Running   0          53m

$ kubectl get po
NAME                        READY   STATUS              RESTARTS   AGE
hostname-5ff7b9fdf6-2psv9   0/1     ContainerCreating   0          1s
hostname-5ff7b9fdf6-dvlvm   1/1     Running             0          19s
hostname-7cf6d977b9-fl8lh   1/1     Running             0          53m
hostname-7cf6d977b9-w4xmj   1/1     Running             0          53m
hostname-7cf6d977b9-wc8gl   1/1     Terminating         0          53m

$ kubectl get po
NAME                        READY   STATUS        RESTARTS   AGE
hostname-5ff7b9fdf6-2psv9   1/1     Running       0          50s
hostname-5ff7b9fdf6-5g94w   1/1     Running       0          32s
hostname-5ff7b9fdf6-dvlvm   1/1     Running       0          68s
hostname-7cf6d977b9-fl8lh   1/1     Terminating   0          54m
hostname-7cf6d977b9-w4xmj   1/1     Terminating   0          54m
hostname-7cf6d977b9-wc8gl   0/1     Terminating   0          54m

$ kubectl get po
NAME                        READY   STATUS    RESTARTS   AGE
hostname-5ff7b9fdf6-2psv9   1/1     Running   0          7m13s
hostname-5ff7b9fdf6-5g94w   1/1     Running   0          6m55s
hostname-5ff7b9fdf6-dvlvm   1/1     Running   0          7m31s
```

롤링 업데이트가 진행되는 도중에 `kubectl get pods`로 파드의 상태를 추적해보았다. 실제로 새로운 파드인 `hostname-5ff7b9fdf6-dvlvm`가 실행되고 이전 버전인 `hostname-7cf6d977b9-wc8gl`가 제거되는 것을 확인할 수 있다. 파드에 종료 시그널을 보내고 일정시간 종료를 기다리기 때문에 모든 파드가 교체되어도 이전 파드가 Terminating 상태로 출력된다.  

![image](https://user-images.githubusercontent.com/44857109/106571254-469c5c80-657a-11eb-9e64-eca4a0a8c1ce.png)

파드가 교체되는 동안 curl을 반복적으로 실행한 결과이다. 업데이트가 진행중인 상황에서 v1과 v2의 어플리케이션이 혼재되어 서비스된다.

```
$ while true; do curl 192.168.219.201:30123; sleep 1; done
...
This is v1 running in pod hostname-7cf6d977b9-wc8gl
This is v2 running in pod hostname-5ff7b9fdf6-dvlvm
This is v1 running in pod hostname-7cf6d977b9-w4xmj
This is v1 running in pod hostname-7cf6d977b9-fl8lh
This is v1 running in pod hostname-7cf6d977b9-w4xmj
This is v2 running in pod hostname-5ff7b9fdf6-dvlvm
This is v1 running in pod hostname-7cf6d977b9-fl8lh
...
```

> Deployment의 파드 템플릿에서 ConfigMap을 참조하는 경우 ConfigMap을 수정해도 업데이트가 트리거되지 않는다. 어플리케이션의 설정을 수정해야 할 때 업데이트를 트리거하려면 새로운 ConfigMap을 생성하고 해당 리소스를 참조하도록 파드 템플릿을 수정하는 것이다.

하나 주목하고 넘어갈점은 이전 버전의 파드를 관리하던 ReplicaSet이 삭제되지 않고 그대로 남아있다는 것이다.

```
$ kubectl get rs
NAME                  DESIRED   CURRENT   READY   AGE
hostname-5ff7b9fdf6   3         3         3       5h7m
hostname-7cf6d977b9   0         0         0       6h1m
```

<br>

### Deployment 롤백
롤백을 시연하기 위해 처음 4번의 요청은 성공하고 다음 요청부터는 오류를 반환하는 `kbzjung359/kia-hostname:v3`를 생성했다. 새로운 버전으로 업데이트해보자.

```
$ kubectl set image deployment hostname nodejs=kbzjung359/kia-hostname:v3
deployment.apps/hostname image updated

$ kubectl rollout status deployment hostname
Waiting for deployment "hostname" rollout to finish: 1 out of 3 new replicas have been updated...
Waiting for deployment "hostname" rollout to finish: 2 out of 3 new replicas have been updated...
Waiting for deployment "hostname" rollout to finish: 1 old replicas are pending termination...
deployment "hostname" successfully rolled out
```

이제 새로운 서비스에 클라이언트가 요청을 보내면 오류를 수신하기 시작한다.

```
$ while true; do curl 192.168.219.201:30123; sleep 1; done
This is v3 running in pod hostname-9c457bc95-x7mj6
This is v3 running in pod hostname-9c457bc95-dvqvv
This is v3 running in pod hostname-9c457bc95-dvqvv
This is v3 running in pod hostname-9c457bc95-t2948
Some internal error has occurred! This is pod hostname-9c457bc95-dvqvv
Some internal error has occurred! This is pod hostname-9c457bc95-dvqvv
This is v3 running in pod hostname-9c457bc95-t2948
This is v3 running in pod hostname-9c457bc95-x7mj6
Some internal error has occurred! This is pod hostname-9c457bc95-t2948
```

사용자가 서버 오류를 계속 경험하게 할 수 없으므로 빠르게 조치를 취해야한다. 다음 섹션에서 자동으로 잘못된 롤아웃을 차단하는 방법을 볼 수 있지만 먼저 수동으로 작업해보자. Deployment는 배포의 마지막 롤아웃을 실행 취소하도록 지시해서 이전에 배포된 버전으로 쉽게 롤백할 수 있다.

```
$ kubectl rollout undo deployment hostname
deployment.apps/hostname rolled back

$ kubectl get po
NAME                        READY   STATUS              RESTARTS   AGE
hostname-5ff7b9fdf6-7c6w9   0/1     ContainerCreating   0          1s
hostname-5ff7b9fdf6-7hkd2   1/1     Running             0          16s
hostname-9c457bc95-dvqvv    1/1     Running             0          7m20s
hostname-9c457bc95-t2948    1/1     Running             0          7m42s
hostname-9c457bc95-x7mj6    1/1     Terminating         0          7m5s

$ kubectl get po
NAME                        READY   STATUS        RESTARTS   AGE
hostname-5ff7b9fdf6-7c6w9   1/1     Running       0          35s
hostname-5ff7b9fdf6-7hkd2   1/1     Running       0          50s
hostname-5ff7b9fdf6-bhdsp   1/1     Running       0          22s
hostname-9c457bc95-dvqvv    1/1     Terminating   0          7m54s
hostname-9c457bc95-t2948    1/1     Terminating   0          8m16s
hostname-9c457bc95-x7mj6    0/1     Terminating   0          7m39s
```

> `rollout undo` 명령은 롤아웃 프로세스가 진행중일 때도 실행할 수 있다. 이미 생성된 새로운 파드는 제거되고 다시 이전 파드로 대체된다.

Deployment는 업데이트가 끝나도 이전 기록(ReplicaSet)을 유지하기 때문에 모든 버전으로 롤백할 수 있다. 다음 명령으로 배포 기록을 출력해볼 수 있다.

```
$ kubectl rollout history deployment hostname
REVISION  CHANGE-CAUSE
1         kubectl create --filename=hostname-dp.yaml --record=true
3         kubectl create --filename=hostname-dp.yaml --record=true
4         kubectl create --filename=hostname-dp.yaml --record=true
```

> 책에서는 CHANGE-CAUSE에 `kubectl set image ...` 명령어가 나오는데 나는 다르게 출력된다. 흠.. 아무튼 Deployment를 생성할 때 `--record` 옵션을 주었기 때문에 변경 원인이 출력되었다.

`rollout undo`명령에서 특정 리비전을 지정해서 해당 버전으로 롤백할 수 있다. v1로 롤백해보자.

```
$ kubectl rollout undo deployment hostname --to-revision=1
deployment.apps/hostname rolled back

$ kubectl get po
NAME                        READY   STATUS        RESTARTS   AGE
hostname-5ff7b9fdf6-7c6w9   1/1     Terminating   0          12m
hostname-5ff7b9fdf6-7hkd2   1/1     Terminating   0          12m
hostname-5ff7b9fdf6-bhdsp   0/1     Terminating   0          11m
hostname-7cf6d977b9-9fb94   1/1     Running       0          24s
hostname-7cf6d977b9-dztlf   1/1     Running       0          52s
hostname-7cf6d977b9-t2zbg   1/1     Running       0          38s

$ curl 192.168.219.201:30123
This is v1 running in pod hostname-7cf6d977b9-t2zbg
```

Deployment에 의해 생성된 모든 ReplicaSet은 전체 리비전 정보를 나타낸다. 각 ReplicaSet은 특정 리비전에 배포의 전체 정보를 저장하기 때문에 수동으로 특정 ReplicaSet을 삭제한다면 안된다.

하지만 너무 오래된 ReplicaSet으로 인해 RS 목록이 복잡해지는 것은 이상적이지 않기 때문에 리비전 기록의 길이를 제한할 수 있다. `revisionHistoryLimit`의 기본값은 2 이기 때문에 일반적으로 현재 및 이전 리비전만 기록된다. 

> 내가 로컬 클러스터에서 실습하고 있는 버전 `v1.20.0`에서는 `revisionHistoryLimit`의 기본값이 3 인 것 같다. 히스토리 기록이 3개 출력된다.

```
$ kubectl rollout history deployment hostname
deployment.apps/hostname
REVISION  CHANGE-CAUSE
3         kubectl create --filename=hostname-dp.yaml --record=true
4         kubectl create --filename=hostname-dp.yaml --record=true
5         kubectl create --filename=hostname-dp.yaml --record=true
```

![image](https://user-images.githubusercontent.com/44857109/106608413-b83ecf80-65a7-11eb-8d0b-5537dfdab11f.png)


<br>

### 롤아웃 속도 제어
롤아웃을 수행할 때 새 파드가 생성되고 이전 파드가 삭제되는 방식은 롤링 업데이트 전략의 두가지 추가 속성을 통해 구성할 수 있다. `maxSurge`, `maxUnavailable` 속성은 롤링 업데이트 중에 한번에 교체되는 파드의 수를 설정한다.

- maxSurge
  - Deployment에 구성된 복제본 수 이상으로 존재하도록 허용하는 파드 인스턴스 수를 결정한다.
  - 백분률 기본값은 25% 이다. 
    - 만약 복제본 수가 4라면 업데이트 중에 동시에 실행되는 파드 인스턴스가 5개를 초과하지 않는다. 
    - 백분률은 반올림된다.
  - 절대값으로도 지정할 수 있다.
- maxUnavailable
  - Deployment에 구성된 복제본 수를 기준으로 사용할 수 없는 파드 인스턴스 수를 결정한다.
  - 백분률 기본값은 25% 이다. 따라서 업데이트 중에 사용 가능한 파드의 인스턴스 수가 75% 이하로 떨어지면 안된다.
    - 만약 복제본 수가 4라면 한 개의 파드만 Terminate 상태로 만들 수 있다. 전체 롤아웃 동안 3개 이상의 파드 인스턴스를 사용할 수 있다.
  - 절대값으로도 지정할 수 있다.

```yaml
spec:
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1 # 절대값으로 지정
      maxUnavailable: 0
```

이전에 실습했던 Deployment는 복제본 수가 3이었기 때문에 최대 4개의 파드가 실행될 수 있도록 허용하고 사용할 수 없는 파드를 허용하지 않기 때문에 항상 3개 이상의 파드를 사용할 수 있어야 했다.

![image](https://user-images.githubusercontent.com/44857109/106611118-ec67bf80-65aa-11eb-8d00-261937886ffb.png)

만약 maxSurge와 maxUnavailable가 보두 1 이었다면 최대 4개의 파드가 실행될 수 있도록 허용하고 항상 2개 이상의 파드가 사용 가능한 상태로 업데이트가 진행될 것이다.

![image](https://user-images.githubusercontent.com/44857109/106611890-d1497f80-65ab-11eb-9fcf-b3c55cf62d7d.png)

<br>

### 롤아웃 프로세스 일시 정지 시키기
이전 어플리케이션의 v3 버전을 수정해서 v4 이미지를 배포해야 하는 상황이라 가정하자. 이전에 했던 방식으로 모든 파드에 대해 롤아웃하는 것은 약간 부담이 될 수 있다. 따라서 v2 파드가 실행되는 상태에서 v4 파드가 옆에서 단일 파드로 실행되고 전체 사용자중 일부만 새로운 버전에 접근하도록 하려한다. 그런 다음 v4가 정상적으로 작동하는 것을 확인하고 모든 파드를 v4 로 교체하는 계획이다.

직접 ReplicaSet, Deployment를 추가, 수정해서 수행할 수있지만 Deployment 자체에서 이러한 기능을 지원한다. 롤아웃을 수행하는 도중에 프로세스를 일시 중지할 수 있다. 이를 통해 나머지 롤아웃을 진행하기 전에 새 버전에서 오작동 있는지 확인할 수 있다.

```
$ kubectl set image deployment hostname nodejs=kbzjung359/kia-hostname:v4
deployment.apps/hostname image updated

$ kubectl rollout pause deployment hostname
deployment.apps/hostname paused

# 롤링 업데이트를 위해 1개의 새 파드가 생성된 상태에서 일시 중지
$ kubectl get po
NAME                        READY   STATUS    RESTARTS   AGE
hostname-5467ccddb6-pt864   1/1     Running   0          28s
hostname-7cf6d977b9-9fb94   1/1     Running   0          52m
hostname-7cf6d977b9-dztlf   1/1     Running   0          52m
hostname-7cf6d977b9-t2zbg   1/1     Running   0          52m

# 일부 사용자는 v4 파드의 응답을 수신한다.
$ while true; do curl 192.168.219.201:30123; done
This is v1 running in pod hostname-7cf6d977b9-t2zbg
This is v1 running in pod hostname-7cf6d977b9-t2zbg
This is v4 running in pod hostname-5467ccddb6-pt864
This is v1 running in pod hostname-7cf6d977b9-9fb94
This is v1 running in pod hostname-7cf6d977b9-dztlf
```

이렇게 하면 `canary release`를 효과적으로 실행할 수 있다. 라나리 릴리즈는 잘못된 버전의 어플리케이션을 출시할 위험을 최소화하는 기술이다. 모든 사람에게 새 버전을 배포하는 대신 소수의 파드만 새 파드로 교체해서 소수의 사용자만 새 버전을 사용하게 하게 한다. 그런 다음 새 버전이 제대로 작동하는지 여부를 확인한 다음 모든 파드에 대해서 롤아웃을 이어서 진행하거나 다시 이전 버전으로 롤백한다.

<br>

이제 v4 파드가 정상적으로 작동하는지 확인했으므로 롤아웃을 재개해보자.

```
$ kubectl rollout resume deployment hostname
deployment.apps/hostname resumed

$ kubectl get po
NAME                        READY   STATUS              RESTARTS   AGE
hostname-5467ccddb6-9dz5w   0/1     ContainerCreating   0          3s
hostname-5467ccddb6-pt864   1/1     Running             0          7m14s
hostname-7cf6d977b9-9fb94   1/1     Terminating         0          59m
hostname-7cf6d977b9-dztlf   1/1     Running             0          59m
hostname-7cf6d977b9-t2zbg   1/1     Running             0          59m
```

<br>

### 잘못된 버전 출시 방지
처음에 Deployment 리소스를 생성했을 때 `minReadySeconds` 속성을 설정했었다. 롤아웃 과정을 확인해보기 위해서 롤아웃 속도를 늦추는 목적으로 사용했지만 실제로 롤링 업데이트를 수행할 때 모든 파드를 한번에 교체하지 않는 목적으로 사용한다. 즉 이 속성의 주요 기능은 그저 재미로 속도를 조절하는 것이 아니라 오작동하는 버전 배포를 방지하는 것이다.

`minReadySeconds`이 설정되면 새 파드가 Readiness Probe에 의해 ready 상태가 되더라도 롤아웃 프로세스를 해당 시간동안 더 기다린다. 파드를 프로덕션에 배포하기 전에 테스트 및 스테이징 환경 모두에서 철저하게 테스트를 하겠지만 이 설정값을 통해서 프로덕션에 버그가 있는 버전을 배포하지 않도록 하는 에어백 역할을 한다.

적절한 Readiness Probe와 minReadySecond을 설정하면 쿠버네티스가 버그가 있는 버전을 배포하지 않도록 할 수 있다.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: hostname
spec:
  replicas: 3
  minReadySeconds: 10
  strategy: # 업데이트 전략 설정
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  selector:
    matchLabels:
      app: hostname
  template:
    metadata:
      name: hostname
      labels:
        app: hostname
    spec:
      containers:
      - image: kbzjung359/kia-hostname:v3
        name: nodejs
        ports:
        - containerPort: 8080
        readinessProbe: # 준비 프로브 설정
          periodSeconds: 1
          httpGet:
            path: /
            port: 8080
```

기존의 Deployment를 업데이트하려면 `kubectl apply` 명령어를 사용한다.

```
$ kubectl apply -f hostname-dp-readiness.yaml
Warning: resource deployments/hostname is missing the kubectl.kubernetes.io/last-applied-configuration annotation which is required by kubectl apply. kubectl apply should only be used on resources created declaratively by either kubectl create --save-config or kubectl apply. The missing annotation will be patched automatically.
deployment.apps/hostname configured
```

> `kubectl apply`는 선언적으로 이전에 생성된 리소스에만 사용해야 하기 때문에 `kubernetes.io/last-applied-configuration` 주석을 사용해야 한다고 경고를 준다.

```
$ while true; do curl 192.168.219.201:30123; sleep 1; done
This is v4 running in pod hostname-5467ccddb6-976rr
This is v4 running in pod hostname-5467ccddb6-pt864
This is v4 running in pod hostname-5467ccddb6-pt864
This is v4 running in pod hostname-5467ccddb6-976rr
This is v4 running in pod hostname-5467ccddb6-976rr
This is v4 running in pod hostname-5467ccddb6-976rr
^C

$ kubectl get po
NAME                        READY   STATUS    RESTARTS   AGE
hostname-5467ccddb6-976rr   1/1     Running   0          21h
hostname-5467ccddb6-9dz5w   1/1     Running   0          21h
hostname-5467ccddb6-pt864   1/1     Running   0          21h
hostname-86b784bfd5-94dcz   0/1     Running   0          2m17s
```

원래대로라면 `hostname-86b784bfd5-94dcz` 는 처음 4개의 요청은 정상적으로 처리하기 때문에 해당 Deployment의 서비스에 요청을 하면 해당 파드로 몇개의 요청이 전달됐어야 했지만 그렇지 않았다. 

실제로 `hostname-86b784bfd5-94dcz` 파드는 처음 4개의 요청을 처리하기 전까지는 정상 동작하기 때문에 롤아웃 프로세스는 새 파드가 작동하는 것을 확인하고 다음 파드를 생성할 수 있었다. 하지만 minReadySeconds를 10초로 설정했기 때문에 바로 다음 파드를 생성하지 않고 10초를 더 기다린 것이다. 그동안 Readiness Porbe가 매초 준비상태를 확인하는 요청을 보내게 되고 그 과정에서 프로브가 실패해서 더이상 롤아웃 프로세스가 진행되지 않았던 것이다.


![image](https://user-images.githubusercontent.com/44857109/106748833-21d2e280-6669-11eb-828f-1258554b3532.png)

minReadySeconds를 설정하지 않고 Readiness Probe만 설정했다면 첫 번째 준비 프로브의 호출이 성공하면 바로 이어서 롤아웃이 진행됐을 것이다. 

Deployment는 `progressDeadlineSeconds`를 설정해서 롤아웃 프로세스가 해당 시간동안 진행할 수 없으면 실패한 것으로 간주해서 자동으로 롤아웃을 중단시킨다. 기본값은 10분이다.

