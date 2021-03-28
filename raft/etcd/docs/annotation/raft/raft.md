# raft
- `etcd/raft/raft.go`

stepLeader -> case MsgTransferLeader (리더가 MsgTransferLeader 메시지를 받은 경우)
// 특정한 팔로워가 자신이 리더가 되고 싶을 때
1. 메시지를 보낸 노드가 Learner라면 메시지 드랍
2. lastLeadTransferee(이전에 메시지를 보낸 노드)가 leadTransferee(지금 처리하고있는 메시지를 보낸 노드)와 같다면 메시지 드랍
3. lastLeadTransferee가 leadTransferee와 다르다면 lastLeadTransferee 초기화
4. leadTransferee가 자신이라면 드랍
5. leadTransferee의 로그 복제 상태가 최신이라면 해당 노드가 선거를 시작하도록 MsgTimeoutNow 메시지 전송. 아니라면 sendAppend를 통해 로그 복제

stepFollower -> case MsgTransferLeader (팔러워가 MsgTransferLeader 메시지를 받은 경우)
1. m.To가 설정되어 있지 않다면 드랍
2. m.To를 r.lead(현재 팔로워가 알고있는 리더)로 설정한뒤 그대로 send


------------------------
stepLeader ->case MsgProp(proposal : 쓰기 연산 제안)
1. 제안된 entries가 없다면 패닉
2. chang configuration에 의해 다음 그룹에서 자신이 제거되어있는 경우 메시지 드랍
3. leadTransferee가 있는 경우(리더자리를 팔로워에게 위임하는 중일 경우) 메시지 드랍
4. 제안된 entries들 중에 confChange를 찾음
4-1. 만약 configChange를 적용할 수 없는 상태라면(alreadyPending, alreadyJoint, wantsLeaveJoint) 해당 엔트리를 MsgNormal로 변경해서 드랍하게 함
4-2. 적용할 수 있다면 pendingConfIndex를 업데이트
5. 제안된 entries를 로그에 추가
6. bcastAppend를 통해 팔로워들에게 엔트리 복제 메시지 전송

stepFollower -> case MsgProp
1. 팔로워 노드가 알고있는 리더가 없다면 드랍
2. 노드의 상태가 disableProposalForwarding이라면 드랍
3.  m.To를 r.lead(현재 팔로워가 알고있는 리더)로 설정한뒤 그대로 send


-------------------------
stepLeader -> case MsgReadIndex : readOnly 요청을 위해 리더에게 리더의 commited index 요청
1. 클러스터에서 투표할 수 있는 맴버가 자신밖에 없다면 바로 resq 메시지를 만들어 전송
2. commit된 entry가 현재 리더의 term이 아니라면 응답 거부
3. readOnly 옵션에 따라 적절히 처리

stepFollower -> case MsgReadIndex
1. 팔로워가 알고있는 리더가 없다면 드랍
2. m.To를 r.lead로 설정한뒤 그대로 send

stepFollower -> case MsgReadIndexResp
1. len(m.Entries) != 1 이라면 respMsg 포멧 에러
2. r.readStates에 m.Index 추가


-------------------------
stepLeader -> case MsgAppResp : appendEntry에 대한 팔로워의 응답
if reject
1. 만약 reject라면 pr.Next를 m.RejectHint, m.LogTerm에 따라 적절히 설정
2. pr.Next를 감소시켜야 한다고 판단되면 감소시키고 probe의 상태를 Replicate에서 내리고 sendAppend 호출해서 엔트리나 스냅샷 전송

if not reject
1. m.Index가 m.Match보다 크다면(팔로워의 로그가 정상적으로 업데이트 되었다면)
2. probe의 상태에 따라 Replicate 상태로 올림
3. 만약 commitedIndex가 업데이트되었다면 bcastAppend로 모든 노드에게 해당 정보 전달, 프로브가 정지된 상태(snapshot 보내는 중, 전송 계층 용량 가득참, 전송이 지연되도록 설정됨)일 때 sendAppend로 해당 정보 전달
4. 프로브의 상태를 업데이트했기 때문에 maybeSendAppend로 추가적으로 보낼 수 있는 엔트리나 스냅샷이 있다면 보냄
5. 메시지를 보낸 노드가 leadTransferee이고 로그가 최신상태라면 MsgTimeoutNow 메시지를 보냄

-------------------------
stepLeader -> case MsgHeartbeatResp
1. probe 상태 업데이트 (RecentActive: true, ProbeSent: false)
2. 프로브가 Replicate 상태이고 전송 용량이 가득찼다면 빈자리를 만듬
3. 해당 노드에게 더 보낼 엔트리가 있다면 snedAppend
4. TODO: readOnlyAdvance 머시기

-------------------------
stepLeader -> case MsgSnapStatus
1. 프로브가 Snapshot 상태가 아니라면 리턴
2. m.Reject라면(Follower가 Snapshot을 처리하다가 오류가 발생한 경우) PendingSnapshot을 0으로 설정
3. 프로브를 Probe상태로 설정
4, ProbeSent를 true로 설정해서 전송을 지연시킴

-------------------------
stepLeader -> case MsgUnreachable TODO: 어떤 상황에 보냄?
1. 프로브가 Replicate 상태라면 Probe상태로 만듬