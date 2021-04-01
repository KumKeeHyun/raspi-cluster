# raspi-cluster
Raspberry pi 4 를 이용해서 클러스터 환경을 공부하는 레포

## Table of Contents
- Kubernetes: 쿠버네티스를 공부하는 곳 
    - Raspberry Pi-4 4개를 이용해서 직접 쿠버네티스 환경 구성, 실습
    - 책이나 아티클을 읽고 정리
        - kubnernetes-in-action
    - TODO: 파드 네트워크 플러그인, 보안 솔루션, 개별 구성 요소 분석 
- Microservice Architecture(MSA): MSA를 공부하는 곳
    - 책이나 아티클을 읽고 정리
    TODO: Building-Microservice 읽기만 하고 정리는 못함, 마이크로서비스 도입, 이렇게 한다(Monolith to Microservice) 책 읽는 중
- Raft Consensus Algorithm: Raft 합의 알고리즘을 공부하는 곳
    - etcd/raft 라이브러리 코드를 분석하고 정리
    - TODO: raft 논문 정리, etcd/raft 라이브러리의 snapshot 생성, 적용 코드 흐름 분석

## OS
Ubuntu Server 20.04.1 LTS
- https://ubuntu.com/download/raspberry-pi

## Component
<img src="https://user-images.githubusercontent.com/44857109/102356152-32af9200-3ff0-11eb-8c17-2e546cf1754a.png" width="70%" height="70%">

- Raspberry pi 4 - RAM 8Gb 3개
- Raspberry pi 4 - RAM 4Gb 1개