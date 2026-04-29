# NFR Requirements Plan — U3 Realtime Transport

**작성일**: 2026-04-26
**대상 단위**: U3 / Realtime Transport (`internal/transport/ws`)
**참조**:
- `requirements.md` v1.1 NFR-1/2/4/6/7
- `aidlc-docs/construction/u3-realtime-transport/functional-design/*.md`
- `aidlc-docs/construction/u2-session-persistence-announce/nfr-requirements/nfr-requirements.md` (참고 — 인접 단위)

> 본 plan은 U3 NFR Requirements의 단일 진실 소스입니다.

---

## 0. NFR 영역 우선순위

U3는 인프라 단위(WebSocket I/O 전담)이므로 **Performance / Concurrency**가 최우선.

| 영역 | 적용 여부 | 비고 |
|---|:---:|---|
| Reliability | 중간 | 클라이언트 끊김 감지·재연결 친화 (BR-U3-RECONNECT) |
| Performance | **상위** | LAN 즉시 반응 + 12명 동접 (NFR-2) |
| Concurrency | **상위** | per-client read/write goroutine + 단일 onEvent 진입점 |
| Maintainability | 중간 | 단위 테스트, 정적 분석, 외부 의존 1개 (gorilla/websocket) |
| Security | 중간 | NFR-4 비공개 정보 라우팅 (BR-U3-VIS-2/3 + LOG-2) |
| Scalability / Availability | **N/A** | 단일 호스트 PC, 단일 게임 (NFR-7) |

---

## 1. 단계 체크리스트

- [x] (1) 영역 우선순위 평가
- [x] (2) 결정 질문 작성 (Q-NFR-U3-1~12)
- [x] (3) plan 문서 작성 (본 파일)
- [x] (4) 사용자 답변 수집 — "승인" (권장 답안)
- [x] (5) 답변 일관성 검증·모호성 해결 — 모호성 없음
- [x] (6) `nfr-requirements.md` 작성 — 영역별 측정 가능 NFR + 트레이드오프 + 추적성 + Build & Test 검증 게이트
- [x] (7) `tech-stack-decisions.md` 작성 — 외부 의존 1개(gorilla/websocket) + 표준 lib 사용 결정
- [x] (8) audit + aidlc-state 갱신, 사용자 승인 게이트

---

## 2. 결정 질문 (Q-NFR-U3-1 ~ Q-NFR-U3-12)

### Q-NFR-U3-1. 외부 의존성 한정

U3에 추가하는 외부 lib는?

- **A.** `github.com/gorilla/websocket` 1개만 (Q-AD-2=A 그대로). 나머지는 Go 표준 lib (`encoding/json`, `log/slog`, `net/http`, `crypto/rand`).
- **B.** + `nhooyr.io/websocket` (대안 — context 친화).
- **C.** 표준 `golang.org/x/net/websocket`만 사용.

[Answer]: A

### Q-NFR-U3-2. SubmitAction → 첫 클라이언트 push 지연 (Performance)

LAN 환경에서 클라이언트가 `submit:vote` 송신 후 다른 클라이언트의 첫 push까지 지연 목표는?

- **A.** **p99 < 200 ms** (12명 LAN, 평시). gorilla/websocket TextMessage write 통상 < 1 ms.
- **B.** p99 < 100 ms (더 엄격).
- **C.** p99 < 500 ms (느슨).

[Answer]: A

### Q-NFR-U3-3. 동시 연결 처리 능력

Hub가 동시에 처리해야 하는 WS 연결 수의 측정 가능 한도는?

- **A.** **12명 PLAYER + 4 PUBLIC = 16 연결** 동시 처리 (요구사항 FR-1.3 + α). 메모리 / goroutine 누수 없이 안정 동작.
- **B.** 30 연결 (확장 마진).
- **C.** 100 연결 (PoC 범위 외).

[Answer]: A

### Q-NFR-U3-4. 단위 테스트 커버리지 목표

`internal/transport/ws` 패키지의 라인 커버리지 목표는?

- **A.** **≥ 80%** — gorilla/websocket의 실제 conn 모킹이 어려운 부분 감안 (하지만 net.Pipe 기반 in-process 테스트로 80% 달성 가능).
- **B.** ≥ 85% — U2와 동일 수준.
- **C.** ≥ 70% — WS 라이브러리 외부 의존 더 보수적.

[Answer]: B

### Q-NFR-U3-5. 정적 분석 / 포맷 게이트

빌드 게이트는?

- **A.** `go vet` 0 issue + `gofmt -l` empty + `go test -race` 통과 (U2와 동일).
- **B.** + `golangci-lint run` (default) 0 issue.
- **C.** A보다 느슨 — vet만.

[Answer]: A

### Q-NFR-U3-6. read/write deadline 정책

연결당 read deadline 30초, write deadline 10초로 확정해도 되나?

- **A.** **30초 read / 10초 write** — FD §4, §6 결정. ping 25초로 갱신.
- **B.** 더 짧게 (15초/5초) — 더 빠른 끊김 감지.
- **C.** 더 길게 (60초/30초) — 느슨.

[Answer]: A

### Q-NFR-U3-7. 메시지 크기 한도

수신 메시지 최대 크기는?

- **A.** **64 KiB**. gorilla `Conn.SetReadLimit(64 << 10)`. 정상 메시지(예: SubmitVote)는 < 1 KiB.
- **B.** 1 MiB (관대).
- **C.** 8 KiB (엄격).

[Answer]: A

### Q-NFR-U3-8. 비공개 정보 라우팅 검증 방법

NFR-4의 핵심 — 비공개 이벤트가 잘못된 클라이언트로 전달되면 안 됨. 검증은?

- **A.** **단위 테스트로 가시성 라우팅 검증** — `VisPublic` / `VisPlayer` / `VisRoleMafia` 각 케이스마다 의도한 클라이언트만 메시지를 받았는지 확인. 모킹 net.Pipe로 충분.
- **B.** + 통합 테스트로 실제 게임 1판 진행하면서 마피아 전용 이벤트가 시민에게 가지 않는지 검증.
- **C.** 테스트 안 함 (코드 리뷰만).

[Answer]: B

### Q-NFR-U3-9. SubmitAction 직렬화 (Concurrency)

여러 클라이언트가 동시 SubmitAction을 보낼 때 직렬화 보장 방식은?

- **A.** **U2 SessionManager의 단일 GM 락이 직렬화 책임을 짐** — Hub는 단순히 forward만, 자체 락 없음. 각 read goroutine이 SubmitAction을 즉시 호출.
- **B.** Hub 자체 입력 큐를 두고 하나의 액터 고루틴이 직렬 처리.
- **C.** Hub가 별도 mutex를 두고 SubmitAction 호출 직전 lock.

[Answer]: A

### Q-NFR-U3-10. graceful shutdown 시간

서버 종료 시 모든 WS 연결을 정리하기까지 허용 시간은?

- **A.** **< 2초** — Close() 호출 후 모든 클라이언트 close + 모든 goroutine 종료. 단위 테스트로 검증.
- **B.** < 5초.
- **C.** 시간 제한 없음 (어차피 호스트 PC 종료).

[Answer]: A

### Q-NFR-U3-11. 메시지 직렬화 결정성

같은 도메인 이벤트는 동일 wire JSON 바이트로 직렬화되어야 하나?

- **A.** **결정적 직렬화 (encoding/json default)** — Go 1.12+ encoding/json은 map 키를 정렬. 디버깅·로그 비교 용이.
- **B.** 비결정적 허용 (성능 이점 없음, 단순화).
- **C.** 별도 정렬 라이브러리.

[Answer]: A

### Q-NFR-U3-12. fuzz / property test

WS 메시지 디코딩 fuzz test 적용?

- **A.** fuzz test 미적용 (PoC 범위 외) — 단위 테스트로 충분.
- **B.** Go 1.18+ 표준 `testing.F`로 incoming envelope fuzz.
- **C.** 외부 fuzz 도구.

[Answer]: A

---

## 3. 산출물 예상

| 파일 | 책임 |
|---|---|
| `nfr-requirements.md` | 6개 영역(Reliability/Performance/Concurrency/Maintainability/Security/Storage) NFR 항목 ID + 측정 가능 한도 + 검증 방법 + 트레이드오프 표 + FR/NFR 추적성 매트릭스 + 검증 게이트 |
| `tech-stack-decisions.md` | gorilla/websocket 단일 외부 의존 + Go 표준 lib + 패키지 레이아웃(`internal/transport/ws/`) + 의존 그래프 |

---

## 4. 사용자 승인 게이트

본 plan과 답변을 검토해 주세요. 각 `[Answer]:` 줄 변경 또는 추가 질문이 있으시면 알려주세요. 모든 답변에 동의하시면 **"완료"** 또는 **"승인"** 으로 응답해 주시면 산출물 2종 생성 후 NFR Design 승인 게이트로 진입합니다.
