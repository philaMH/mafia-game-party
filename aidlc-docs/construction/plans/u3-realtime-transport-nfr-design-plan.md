# NFR Design Plan — U3 Realtime Transport

**작성일**: 2026-04-26
**대상 단위**: U3 / Realtime Transport (`internal/transport/ws`)
**참조**:
- `nfr-requirements.md` (NFR-U3-R/P/C/M/S/G)
- `tech-stack-decisions.md`
- `functional-design/*.md`
- U2 NFR Design 참고: `aidlc-docs/construction/u2-session-persistence-announce/nfr-design/*.md`

> 본 plan은 U3 NFR Design의 단일 진실 소스입니다.

---

## 0. NFR Design의 목적

NFR Requirements가 정한 측정 가능한 한도(예: p99 < 200ms, 라우팅 정확성)를 만족시키기 위한 **기술 패턴**과 **논리 컴포넌트**를 결정합니다.

---

## 1. 단계 체크리스트

- [x] (1) NFR Req → 적용 패턴 매핑 분석
- [x] (2) 결정 질문 작성 (Q-NFRD-U3-1~8)
- [x] (3) plan 문서 작성 (본 파일)
- [x] (4) 사용자 답변 수집 — "승인" (권장 답안)
- [x] (5) 답변 일관성 검증 — 모호성 없음
- [x] (6) `nfr-design-patterns.md` — 패턴 카탈로그 + 적용 위치 + 안티패턴
- [x] (7) `logical-components.md` — LC-U3-1~11 + 패키지 레이아웃 + 책임 매트릭스
- [x] (8) audit + aidlc-state 갱신, 사용자 승인 게이트

---

## 2. 결정 질문 (Q-NFRD-U3-1 ~ Q-NFRD-U3-8)

### Q-NFRD-U3-1. ClientRegistry 동기화 방식

`map[ClientID]*Client` + `map[PlayerID]*Client`의 동시 접근 보호?

- **A.** **단일 `sync.RWMutex`** + read 경로(라우팅 시 대상 조회)는 RLock, write 경로(Register/Unregister)는 Lock. 단순·정확.
- **B.** `sync.Map` 사용. 장점: lock-free read. 단점: 양쪽 인덱스 일관성 보장이 까다로움.
- **C.** Hub 자체 락(actor 모델)에 위임 — registry 자체 락 없음.

[Answer]: A

### Q-NFRD-U3-2. onEvent 호출 시 가시성 라우팅 — 락 보유 패턴

SessionManager의 락 안에서 호출되는 onEvent가 ClientRegistry RLock을 획득해야 하나? 데드락 위험은?

- **A.** **RLock 획득 + 빠르게 enqueue 후 RUnlock**. SessionManager 락 → registry RLock 한 방향 — 데드락 없음 (registry → SessionManager 호출 경로 없음).
- **B.** registry 스냅샷을 미리 복사 (slice clone) 후 락 해제 → enqueue. 추가 메모리 비용.
- **C.** 락 없이 atomic.Pointer로 registry 교체 (copy-on-write).

[Answer]: A

### Q-NFRD-U3-3. SubmitAction 에러의 Subscribe 핸들러 무경로 처리

U2 `SubmitAction`은 에러 시 `outs[0]`에 announcement를 담아 반환하지만 Subscribe 핸들러는 호출 안 함 (state 변경 없으므로). Hub는 어떻게 송신자에게만 에러 알림?

- **A.** **read goroutine이 SubmitAction을 직접 호출하고 반환된 `outs`/`err`을 자체 처리** — `outs[0].Announcement`가 있으면 송신자에게만 announce 송신, 추가로 `error{code, message}`도 송신자에게.
- **B.** Hub가 send에러도 onEvent를 통해 받도록 U2 인터페이스 변경.
- **C.** 에러는 wire `error` 메시지 1종으로만 — announcement는 무시.

[Answer]: A

### Q-NFRD-U3-4. write goroutine의 Out 채널 close 패턴

Unregister가 close(c.Out)을 호출하면 다른 곳에서 enqueue 시도 시 panic. 가드 패턴?

- **A.** **Client에 `closed atomic.Bool` 플래그 + enqueue 시 RLock 보호 하에 closed 확인** — closed면 enqueue drop. close(c.Out)은 closed=true 설정 후 호출.
- **B.** select default branch만 사용 — enqueue 실패는 무조건 drop, panic은 recover로 처리.
- **C.** 채널 close 안 함 — write goroutine 자체 ctx.Done()으로 종료.

[Answer]: C

### Q-NFRD-U3-5. ID 생성 — 클라이언트 ID 길이

ClientID 길이는?

- **A.** **8 byte → hex16** (16자리). U2의 GameID/PlayerID와 동일 — 디버깅 일관성.
- **B.** 4 byte → hex8 (충분, 메모리 절약).
- **C.** UUID v4 (36자리, 외부 lib 추가 — NFR-U3-M4 위반 가능).

[Answer]: A

### Q-NFRD-U3-6. SessionManager.Snapshot() 추가 vs Hub 자체 마피아 캐시

VisRoleMafia 라우팅은 살아있는 마피아 PlayerID 목록이 필요. 어디서 얻나?

- **A.** **U2 SessionManager 인터페이스에 `Snapshot() game.State` 추가** — Hub의 onEvent에서 락 없이 호출 (SessionManager가 락 안에서 반환). 단일 진실 소스.
- **B.** Hub가 마지막 본 State를 캐시 (이벤트 수신 시 갱신). 동시성 위험·복잡.
- **C.** Hub가 수신한 GameStarted/Eliminated 이벤트를 추적해 자체 마피아 ID set 유지.

[Answer]: A

### Q-NFRD-U3-7. UpgradeHandler 노출 방식

U4 HTTP layer가 어떻게 Hub에 conn을 전달?

- **A.** **Hub가 `UpgradeHandler() http.HandlerFunc` 메서드 제공** — U4는 `mux.HandleFunc("/ws", hub.UpgradeHandler())`만 호출. 캡슐화.
- **B.** U4가 직접 `websocket.Upgrader.Upgrade` 호출 후 `hub.Register(conn)`. Hub는 Upgrader 비의존.
- **C.** 둘 다 노출 — 호출자 선택.

[Answer]: A

### Q-NFRD-U3-8. 통합 테스트 패턴

NFR-U3-S1(비공개 라우팅)과 NFR-U3-P1(push 지연)을 통합 테스트로 검증할 때 진짜 TCP를 쓰나?

- **A.** **`net.Pipe` 기반 in-memory `*websocket.Conn`** — `httptest.NewServer` + `websocket.Dialer`로 실제 WS 핸드셰이크 통과. CI 친화적, race free.
- **B.** `httptest.NewServer` + 클라이언트 측 raw HTTP Upgrade 직접 작성.
- **C.** 실제 OS 소켓 + localhost 포트 — flaky 가능.

[Answer]: A

---

## 3. 산출물 예상

| 파일 | 책임 |
|---|---|
| `nfr-design-patterns.md` | P-U3-1~10 패턴 (예: ClientRegistry RWMutex, write goroutine 단독 ctx.Done, panic recover, prepared JSON, gracefully close) + 안티패턴 |
| `logical-components.md` | LC-U3-1~N 카탈로그 + 패키지 파일 레이아웃 + NFR Req ↔ LC 책임 매트릭스 + import cycle 분석 |

---

## 4. 사용자 승인 게이트

본 plan과 답변을 검토해 주세요. 변경이 필요하면 답을 수정해 알려주시고, 모든 답변에 동의하시면 **"완료"** 또는 **"승인"** 으로 응답해 주세요.
