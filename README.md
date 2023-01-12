# raspi-cluster
Raspberry-pi-4를 이용해서 클러스터 환경을 공부하는 레포

## Table of Contents
- kubernetes: Raspberry-pi-4 4개를 이용해서 직접 쿠버네티스 클러스터 구성, 실습 
    - [docs](./kubernetes/docs): 아티클 읽고 정리
        - Understanding Kubernetes Networking
    - kubnernetes-in-action: 쿠버네티스 인 액션 책 읽고 정리 
- microservice: MSA 공부
    - [docs](./microservice/docs): 아티클 읽고 정리
        - A brief overview and why you should use it in your next project
        - Effective Microservices 10 best practices
        - Microservice Communication Styles
- raft: Raft 합의 알고리즘 공부
    - [etcd](./raft/etcd): etcd의 raft 라이브러리 소스 코드를 분석하고 정리
- kafka: 카프카 공부
    -  kafka-the-definitive-guide: 책 읽고 정리
    - kafka-streams-and-ksqldb: 책 읽고 정리
- golang: 이질적이지만 고랭 언어 공부 

## OS
Ubuntu Server 20.04.1 LTS
- https://ubuntu.com/download/raspberry-pi

## Component
<img src="https://user-images.githubusercontent.com/44857109/102356152-32af9200-3ff0-11eb-8c17-2e546cf1754a.png" width="70%" height="70%">

- Raspberry pi 4 - RAM 8Gb 3개
- Raspberry pi 4 - RAM 4Gb 1개