# Chapter 9. Deployments: updating applications declaratively

<!--ts-->
  - [9.1 Updating Applications Running in Pods](#9.1-Updating-Applications-Running-in-Pods)
    - [기존 파드를 모두 삭제](#기존-파드를-모두-삭제)
    - [새로운 파드를 한번에](#새로운-파드를-한번에)
    - [점진적으로 새로운 파드 생성](#점진적으로-새로운-파드-생성)
  - [9.2 Performing an Automatic Rolling Update with a ReplicationController](#9.2-Performing-an-Automatic-Rolling-Update-with-a-ReplicationController)
    - [rolling-update가 폐기된 이유](#rolling-update가-폐기된-이유)
  - [9.3 Using Deployments for Updating Apps Declaratively](#9.3-Using-Deployments-for-Updating-Apps-Declaratively)
    - [](#)
    - [](#)
    - [](#)
    - [](#)
    - [](#)
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

## Deployment 업데이트
ReplicationController를 사용해서 업데이트를 수행할 때는 수행해야 하는 작업을 모두 명시적으로 지시해야 했다. 또한 rolling-update 명령은 새로운 RC 이름을 지정하고 모든 kubectl가 작업을 끝낼때까지 기다려야 했다. Deployment는 업데이트를 위해 리소스에 정의된 파드 템플릿을 수정하면 쿠버네티스가 알아서 모든 작업을 수행한다.  

업데이트 방법은 `RollingUpdate`와 `Recreate`가 있다. `Recreate`는 RC의 파드 템플릿을 수정하고 모든 파드를 삭제하는 것과 유사하게 모든 이전 파드를 삭제하고 새로운 파드를 생성하는 전략이다. `Recreate`는 새로운 파드가 생성되기 전에 모든 이전 파드를 삭제하기 때문에 어플리케이션이 여러 버전을 병렬로 실행하는 것을 지원하지 않는다. 새 버전을 시작하기 전에 이전 버전을 완전히 중지해야 하는 경우 사용하고 서비스를 완전히 사용할 수 없게되는 다운 타임이 포함된다.

`RollingUpdate`는 이전 파드를 하나씩 제거하는 동시에 새 파드를 생성한다. 전체 프로세스에서 어플리케이션을 계속 사용할 수 있도록 유지하고 요청을 처리할 수 있는 처리량이 낮아지지 않도록 한다. 이전 버전과 새 버전이 동시에 실행될 수 있는 경우 사용한다.

