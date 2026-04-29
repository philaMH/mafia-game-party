# Functional Design Plan — U3 Realtime Transport

**작성일**: 2026-04-26
**대상 단위**: U3 / Realtime Transport (`internal/transport/ws`)
**컴포넌트**: C6 WSHub
**참조**:
- `aidlc-docs/inception/application-design/unit-of-work.md` §3
- `aidlc-docs/inception/application-design/component-methods.md` C6
- `aidlc-docs/inception/application-design/component-dependency.md` §2.2 (비동기 wire), §4 (잠정 와이어 포맷)
- `aidlc-docs/inception/application-design/unit-of-work-story-map.md` (FR/NFR ↔ U3)
- U2 공개 API: `aidlc-docs/construction/u2-session-persistence-announce/code/u2-public-api.md`
- U1 공개 API: `aidlc-docs/construction/u1-game-core/code/u1-public-api.md`

> 본 plan은 U3 Functional Design의 단일 진실 소스입니다.

---

## 0. 단위 컨텍스트 분석

**책임**: 다중 WebSocket 클라이언트 연결 관리 + 도메인 이벤트의 가시성 정책에 따른 라우팅 + 클라이언트 입력의 SessionManager 위임.

**비책임**: HTTP 라우팅 (U4), 비즈니스 규칙 (U1/U2), TTS 발화 (U5).

**입력**:
- U2 SessionManager의 `Subscribe(EventHandler)` 콜백 — `EventOut{Envelope, Announcement}` 수신
- 클라이언트 WebSocket 메시지 (JSON 와이어 포맷)

**출력**:
- 각 클라이언트로의 JSON 메시지 push
- SessionManager 메서드 호출 (`CreateSession`, `JoinPlayer`, `ResumePlayer`, `StartGame`, `SubmitAction`)

**외부 의존**: `github.com/gorilla/websocket` (Q-AD-2=A, 신규 직접 의존 1개).

**Primary FR/NFR**: FR-1.1 LAN URL, FR-1.2 재연결, NFR-2 LAN 즉시 반응 + 12명 동접, NFR-1 클라이언트 재연결 시 화면 자동 복원.

---

## 1. 단계 체크리스트

- [x] (1) 단위 컨텍스트·책임·비책임 정의
- [x] (2) 결정 질문 작성 (Q-FD-U3-1~15)
- [x] (3) plan 문서 작성 (본 파일)
- [x] (4) 사용자 답변 수집 — "권장으로 승인"
- [x] (5) 답변 일관성 검증·모호성 해결 — 모호성 없음
- [x] (6) `domain-entities.md` 작성 — Client/ClientKind/와이어 메시지 타입 + ClientRegistry
- [x] (7) `business-logic-model.md` 작성 — Register/Unregister/Dispatch/Read 흐름 + Subscribe 핸들러 통합
- [x] (8) `business-rules.md` 작성 — 가시성 라우팅·재연결·송신 큐·에러 매핑
- [x] (9) audit + aidlc-state 갱신, 사용자 승인 게이트

---

## 2. 결정 질문 (Q-FD-U3-1 ~ Q-FD-U3-12)

> 각 질문은 다중 선택 형식. `[Answer]: <문자>` 형태로 답해 주세요.

### Q-FD-U3-1. 클라이언트 종류 (ClientKind)

게임 wire format에서 클라이언트를 어떻게 구분하나요?

- **A.** `Public`(공용 화면)과 `Player`(개인 화면) 2종. Public은 PlayerID 없음, Player는 토큰 인증 후 PlayerID 보유. (component-methods.md 기본안)
- **B.** `Public` + `Player` + `Host`(호스트는 별도 권한 채널) 3종.
- **C.** 단일 `Client`(역할 구분 없음) — 메시지 페이로드로 분기.

[Answer]: A

### Q-FD-U3-2. WS 핸드셰이크 인증 흐름

플레이어 클라이언트가 처음 접속할 때 `JoinPlayer`/`ResumePlayer`를 호출하는 시점은?

- **A.** WebSocket 업그레이드 직후 첫 메시지로 `{type:"join", name:"..."}` 또는 `{type:"resume", token:"..."}` 전송 → Hub가 SessionManager 호출 후 결과를 응답으로 push. (가장 단순, 본 PoC 권장)
- **B.** HTTP `/api/join` REST 엔드포인트에서 토큰 발급 → 클라이언트가 토큰을 쿼리스트링으로 첨부해 `/ws?token=...`로 업그레이드.
- **C.** WebSocket 서브프로토콜 헤더로 토큰 전달 (RFC 6455 Sec-WebSocket-Protocol).

[Answer]: A

### Q-FD-U3-3. 와이어 메시지 봉투(envelope) 구조

서버↔클라이언트 메시지 공통 구조는?

- **A.** `{type: string, ...payload}` 평탄 구조 — type별로 필드 직접 (component-dependency.md 잠정안). JSON tag 결정성을 위해 `omitempty` 사용.
- **B.** `{type: string, data: object, id?: string}` 중첩 구조 — request/response correlation을 위해 `id` 필드.
- **C.** Protocol Buffers / MessagePack — 바이너리 효율.

[Answer]: A

### Q-FD-U3-4. 가시성 → 클라이언트 라우팅

`game.EventEnvelope.Visibility`가 라우팅에 어떻게 반영되나요?

- **A.** `VisPublic` → 모든 Public + 살아있는 Player로 push. `VisPlayer` → `Envelope.PlayerID` 1인의 Player에게만. `VisRoleMafia` → 살아있는 마피아 Player 모두에게 (Envelope.PlayerID 무시). (U1 public API §8 가시성 표 그대로)
- **B.** `VisPublic`은 Public만, Player에게는 보내지 않음 (Player는 자기 정보만 받음).
- **C.** Hub가 자체 ACL을 갖고 SessionManager의 가시성을 무시.

[Answer]: A

### Q-FD-U3-5. 죽은 플레이어 라우팅

사망한 Player의 클라이언트는 `VisPublic` 메시지를 계속 받나요?

- **A.** 받음 — 화면에는 "사망" 표시지만 진행 상황은 관전 가능. 시민/관전자 경험 단순화.
- **B.** 받지 않음 — 즉시 차단, 게임 종료 후 GameEnded만 전송.
- **C.** 받지 않음 — 단, 호스트는 받음.

[Answer]: A

### Q-FD-U3-6. 비공개 이벤트 안내(Announcement) 처리

`announce.Announcement.ForPublicOnly == true`인 안내가 Player 채널에 도달하면?

- **A.** Hub가 자동으로 PlayerView 클라이언트에는 송신하지 않음 (필터링) — Public 화면만 자막/TTS 수신. (FD U2 BR-U2-CAT-8과 일치)
- **B.** 모든 클라이언트에 동일하게 송신 — 클라이언트가 자체 필터.
- **C.** 호스트 Player에게만 추가 송신.

[Answer]: A

### Q-FD-U3-7. 송신 큐 / 백프레셔

각 클라이언트의 송신 채널 버퍼와 느린 클라이언트 처리?

- **A.** 클라이언트당 버퍼 16개 채널 + 채널 가득 차면 해당 클라이언트만 강제 disconnect (Hub는 영향 없음). gorilla/websocket의 표준 패턴.
- **B.** 무한 슬라이스 큐 (메모리 위험).
- **C.** 동기 write — 한 클라이언트가 늦으면 Dispatch 전체 지연.

[Answer]: A

### Q-FD-U3-8. ping/pong 하트비트

클라이언트 끊김 감지 방법?

- **A.** 30초 read deadline + 25초마다 서버가 ping 송신 + pong 응답으로 deadline 갱신. read timeout 시 Unregister.
- **B.** 클라이언트가 5초마다 keepalive 메시지 송신, 15초 미수신 시 disconnect.
- **C.** 하트비트 없음 — TCP 끊김에 의존.

[Answer]: A

### Q-FD-U3-9. 동시 입력 처리

같은 PlayerID의 두 클라이언트(예: 데스크톱 + 모바일)가 동시 접속하면?

- **A.** 마지막 connect-wins — 새 연결이 들어오면 기존 연결 강제 종료 + 새 클라이언트가 ResumePlayer 흐름. (재연결 친화적)
- **B.** 첫 연결만 유효 — 두 번째 연결은 거부.
- **C.** 둘 다 유지 — 두 클라이언트 모두 같은 PlayerID로 메시지 수신/송신.

[Answer]: A

### Q-FD-U3-10. SessionManager 통합 패턴

Hub가 SessionManager의 이벤트를 받는 방식?

- **A.** `mgr.Subscribe(handler)`로 1개 핸들러 등록. 핸들러는 `EventOut`을 받아 가시성 분류 + 라우팅. SessionManager 락 안에서 호출되므로 핸들러는 빠르게 클라이언트 채널에 enqueue 후 즉시 반환.
- **B.** SessionManager가 push하는 채널을 Hub가 polling.
- **C.** 매 클라이언트 메시지마다 Hub가 SessionManager.Snapshot을 polling.

[Answer]: A

### Q-FD-U3-11. 입력 라우팅 (클라이언트 → SessionManager)

클라이언트가 `{type:"submit:vote", target:"p2"}`를 보냈을 때 Hub가 어떻게 처리?

- **A.** Hub가 메시지 type별로 `game.Action` 객체를 빌드 → `mgr.SubmitAction(ctx, action)` 호출. 에러 응답은 송신자에게만 push (announce.RenderError 결과).
- **B.** Hub가 raw JSON을 SessionManager에 전달 → SessionManager가 자체 디코딩.
- **C.** Hub가 별도 워커 큐로 보내고 백그라운드 처리.

[Answer]: A

### Q-FD-U3-12. 호스트 컨트롤

호스트만 발행 가능한 액션(`StartGame`, `ForceEndGame`, `EndDiscussionEarly` 등)은 어떻게 인증?

- **A.** Hub가 클라이언트의 PlayerID와 SessionManager에 알려진 HostID를 비교 — 불일치 시 즉시 거부 (SessionManager까지 가지 않음). 권한 체크 1차 게이트.
- **B.** SessionManager가 단독으로 권한 체크 — Hub는 통과만 시킴 (현 U2 구현은 이미 이렇게 동작).
- **C.** 별도 호스트 비밀번호 인증.

[Answer]: B

### Q-FD-U3-13. 와이어 프로토콜 버전

장기적으로 클라이언트/서버 호환성을 어떻게?

- **A.** `protocolVersion` 필드를 첫 메시지에 포함 — 서버가 거부 가능. 본 PoC는 `"v1"` 고정.
- **B.** 버전 필드 없음 — 코드 변경 시 클라이언트/서버 동시 배포 (단일 바이너리이므로 단순).
- **C.** URL 경로(`/ws/v1`)로 분리.

[Answer]: B

### Q-FD-U3-14. 메시지 로깅

Hub가 송수신 메시지를 로그로 남기나?

- **A.** 디버그 레벨에서만 (운영 환경 OFF). slog로 type만 기록 (페이로드 내용은 비밀 정보 포함 가능 — 토큰/역할 등).
- **B.** 모든 메시지 INFO 레벨 (디버깅 친화적이지만 민감 정보 노출).
- **C.** 로그 없음.

[Answer]: A

### Q-FD-U3-15. 재연결 시 자기 화면 복구

`ResumePlayer` 성공 후 클라이언트는 어떻게 현재 게임 상태를 받나?

- **A.** Hub가 SessionManager.ResumePlayer로부터 받은 `JoinResult{CurrentState, YourRole, ...}`을 단일 `{type:"snapshot", state:..., your:...}` 메시지로 즉시 push. 그 이후 새 이벤트 stream 합류.
- **B.** 클라이언트가 재연결 후 다시 모든 이벤트 history를 받음 (리플레이).
- **C.** 클라이언트가 별도 REST `/api/state`를 폴링.

[Answer]: A

---

## 3. 산출물 예상

| 파일 | 책임 |
|---|---|
| `domain-entities.md` | Client / ClientKind / ClientID / wire 메시지 타입 (incoming + outgoing) / Hub 인터페이스 / ClientRegistry |
| `business-logic-model.md` | Register/Unregister 흐름, Read goroutine, Write goroutine + 송신 채널, Subscribe 핸들러, SubmitAction 흐름, ping/pong, 시퀀스 다이어그램 |
| `business-rules.md` | 가시성 라우팅 BR-U3-VIS, 재연결 BR-U3-RECONNECT, 송신 큐·백프레셔 BR-U3-QUEUE, 인증 BR-U3-AUTH, 와이어 포맷 BR-U3-WIRE, 에러 매핑 BR-U3-ERR, 추적성 (FR-1.1/1.2, NFR-1/2) |

---

## 4. 사용자 승인 게이트

본 plan과 답변을 검토해 주세요. 각 `[Answer]:` 줄을 변경하시려면 해당 문자를 수정하시고, 그 외 일관성·추가 질문이 있으시면 알려주세요. 모든 답변에 동의하시면 **"완료"** 또는 **"승인"** 으로 응답해 주시면 plan에 따라 산출물 3종을 생성하고 다음 단계인 NFR Requirements로 진입하기 위한 추가 승인 게이트를 제시합니다.
