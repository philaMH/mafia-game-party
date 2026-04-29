# NFR Requirements — U3 Realtime Transport

**작성일**: 2026-04-26
**문서 버전**: 1.0
**참조**: `requirements.md` v1.1 NFR-1/2/4/6/7, `construction/u3-realtime-transport/functional-design/*.md`, `plans/u3-realtime-transport-nfr-requirements-plan.md`

---

## 1. NFR 영역별 요구사항

### 1.1 Reliability (NFR-1)

| ID | 요구사항 | 측정 가능 한도 | 검증 방법 |
|---|---|---|---|
| NFR-U3-R1 | Read deadline + ping/pong로 끊긴 연결 자동 감지 후 Unregister | Pong 미수신 후 ≤ 31초 내 정리 | 통합 테스트 (느린 클라이언트 시뮬) |
| NFR-U3-R2 | last-connect-wins 재연결 정책 (BR-U3-RECONNECT-1) | 새 resume 후 1초 내 기존 클라이언트 강제 close | 단위 테스트 |
| NFR-U3-R3 | 재연결 직후 snapshot 메시지 1회 push (BR-U3-RECONNECT-3) | 100% (수신 메시지에 type="snapshot") | 단위 테스트 |
| NFR-U3-R4 | graceful shutdown — Close() 호출 후 모든 클라이언트 close + 모든 goroutine 종료 | **< 2초** (Q-NFR-U3-10=A) | 단위 테스트 (timeout) |
| NFR-U3-R5 | EventHandler / SubmitAction 호출 panic은 해당 클라이언트만 disconnect, Hub 정상 동작 | panic 1회 후 다른 클라이언트 정상 송수신 | 단위 테스트 |

### 1.2 Performance (NFR-2)

| ID | 요구사항 | 측정 가능 한도 | 검증 방법 |
|---|---|---|---|
| NFR-U3-P1 | SubmitAction 수신 → 다른 클라이언트의 첫 push까지 지연 | **p99 < 200 ms** (12 PLAYER LAN) | Go benchmark + net.Pipe |
| NFR-U3-P2 | 동시 처리 가능한 WS 연결 수 | **≥ 16 (12 PLAYER + 4 PUBLIC)** 안정 동작 | 통합 테스트 (16 conn 동시) |
| NFR-U3-P3 | 단일 GameStarted 이벤트 (12 RoleRevealedToPlayer + 1 GameStarted + 1 PhaseChanged ≈ 14 메시지) 모든 대상에 push 시간 | < 50 ms (LAN, 16 연결) | 측정 |
| NFR-U3-P4 | 채널 가득 (16 backlog) 발생 빈도 | 정상 트래픽에서 0회 | 통합 테스트 |
| NFR-U3-P5 | onEvent 호출당 SessionManager 락 점유 추가 시간 | < 5 ms (가시성 분류 + N enqueue) | 측정 |

### 1.3 Concurrency

| ID | 요구사항 | 측정 가능 한도 | 검증 방법 |
|---|---|---|---|
| NFR-U3-C1 | SubmitAction 직렬화 책임은 U2 SessionManager 단일 GM 락에 위임 (Q-NFR-U3-9=A) | Hub 자체 락 0개 (registry mutex 제외) | 코드 리뷰 |
| NFR-U3-C2 | 데이터 레이스 미발생 | `go test -race` 통과 | CI 게이트 |
| NFR-U3-C3 | 클라이언트당 read goroutine 1 + write goroutine 1 — 모든 conn write는 writeLoop에서만 | gorilla/websocket single-writer 요구 만족 | 코드 리뷰 |
| NFR-U3-C4 | onEvent (SessionManager 락 안에서 호출) 내부에서 conn write 호출 금지 | 코드상 enqueue만 — 정적 검증 | 코드 리뷰 + 단위 테스트 |

### 1.4 Maintainability (NFR-6)

| ID | 요구사항 | 측정 가능 한도 | 검증 방법 |
|---|---|---|---|
| NFR-U3-M1 | 단위 테스트 라인 커버리지 | **≥ 85%** (Q-NFR-U3-4=B) | `go test -cover` |
| NFR-U3-M2 | 정적 분석 통과 | `go vet` 0 issue + `gofmt -l` empty (Q-NFR-U3-5=A) | CI 게이트 |
| NFR-U3-M3 | godoc 주석 — 모든 공개 식별자 (Hub, ClientKind, 메시지 타입) | revive 룰 | 코드 리뷰 |
| NFR-U3-M4 | 외부 의존성 한정 | `gorilla/websocket` v1 1개 (Q-NFR-U3-1=A) | `go list -m all` |
| NFR-U3-M5 | 와이어 메시지 타입은 `protocol.go` 1개 파일에 집중 — wire 변경 시 단일 진실 소스 | 코드 리뷰 | — |
| NFR-U3-M6 | JSON 결정적 직렬화 (Q-NFR-U3-11=A) | encoding/json default | 코드 리뷰 |

### 1.5 Security (NFR-4)

| ID | 요구사항 | 측정 가능 한도 | 검증 방법 |
|---|---|---|---|
| NFR-U3-S1 | 비공개 정보 라우팅 정확성 (BR-U3-VIS-2/3) | VisPlayer 이벤트가 다른 PlayerID 클라이언트에 송신 0건 | 단위 테스트 + 통합 테스트 (Q-NFR-U3-8=B) |
| NFR-U3-S2 | DEBUG 로그에 토큰·역할·키워드 미기록 (BR-U3-LOG-2) | 로그 grep으로 키워드 없음 검증 | 코드 리뷰 |
| NFR-U3-S3 | 수신 메시지 크기 한도 | **64 KiB** `Conn.SetReadLimit` (Q-NFR-U3-7=A) | 단위 테스트 (큰 메시지 reject) |
| NFR-U3-S4 | Read deadline 30초 / Write deadline 10초 (Q-NFR-U3-6=A) | 코드상 명시 | 코드 리뷰 |

### 1.6 Storage / Concurrency 자원

| ID | 요구사항 | 측정 가능 한도 | 검증 방법 |
|---|---|---|---|
| NFR-U3-G1 | 클라이언트당 송신 채널 버퍼 | **16 슬롯** (BR-U3-QUEUE-1) | 코드 리뷰 |
| NFR-U3-G2 | Goroutine 누수 0 — 클라이언트 disconnect 후 read+write goroutine 정확히 종료 | 단위 테스트 (`runtime.NumGoroutine()` 비교) | 단위 테스트 |
| NFR-U3-G3 | 메모리 누수 — 1000회 connect/disconnect 반복 후 메모리 증가 < 1 MiB | (옵션) Build & Test 단계 |  |

---

## 2. 트레이드오프 결정

| 트레이드오프 | 본 단위의 결정 | 근거 |
|---|---|---|
| 라이브러리 선택 (gorilla/websocket vs nhooyr/websocket vs std) | **gorilla/websocket** | Q-AD-2=A, Go 생태계 표준, 학습 비용 0 |
| 봉투 구조 (평탄 vs 중첩) | **평탄** `{type, ...}` | Q-FD-U3-3=A, 클라이언트 디코딩 단순 |
| 백프레셔 처리 (블록 vs disconnect) | **disconnect** | Q-FD-U3-7=A, Hub 전체 영향 회피 |
| 권한 체크 위치 (Hub vs SessionManager) | **SessionManager 단독** | Q-FD-U3-12=B, 단일 진실 소스, U2 이미 구현 |
| 동시 접속 정책 (last-wins vs first-wins) | **last-wins** | Q-FD-U3-9=A, 재연결 친화 (네트워크 끊김 흔함) |
| Hub 자체 락 도입 vs 위임 | **위임** | Q-NFR-U3-9=A, U2 GM 락이 이미 직렬화 보장 |

---

## 3. 추적성 (FR/NFR ↔ U3 NFR)

| 출처 | 본 문서 항목 |
|---|---|
| NFR-1 (안정성·복원) | NFR-U3-R1~R5 |
| NFR-2 (성능) | NFR-U3-P1~P5 |
| NFR-4 (비공개·LAN 한정) | NFR-U3-S1~S4 |
| NFR-6 (도메인 분리·유지보수성) | NFR-U3-M1~M6 |
| NFR-7 (외부 서비스 0) | NFR-U3-M4 (gorilla/websocket 1개) |
| FR-1.1 (LAN URL + 단일 호스트) | NFR-U3-P2 |
| FR-1.2 (재연결) | NFR-U3-R2, R3 |
| FR-2.3 (역할 비공개) | NFR-U3-S1, S2 |

---

## 4. 검증 게이트 (Build & Test 단계)

다음 모든 항목이 통과해야 U3가 출하 가능:

1. ✅ `go vet ./internal/transport/ws/...` 0 issue
2. ✅ `gofmt -l ./internal/transport/ws/` empty
3. ✅ `go test -race ./internal/transport/ws/...` 통과 (NFR-U3-C2)
4. ✅ `go test -coverprofile=...` 라인 ≥ **85%** (NFR-U3-M1)
5. ✅ Goroutine 누수 검증 단위 테스트 통과 (NFR-U3-G2)
6. ✅ 비공개 라우팅 단위 테스트 (VisPublic/Player/RoleMafia 각각) + 통합 테스트 (NFR-U3-S1)
7. ✅ Read limit 검증 단위 테스트 (NFR-U3-S3)
8. ✅ graceful shutdown 단위 테스트 < 2초 (NFR-U3-R4)
9. ✅ `go list -m all` — 직접 의존 1개(`gorilla/websocket`) 외 추가 lib 없음

---

## 5. 명시적 비-요구사항 (Non-Goals)

- **Scalability**: 단일 호스트 PC, 단일 게임. 멀티 인스턴스 / sharding / horizontal pod scaling 미지원.
- **Availability SLA**: 직접 책임 없음. NFR-U3-R4 graceful shutdown만 보장.
- **TLS / wss://**: 사내 LAN 가정으로 평문 ws:// 허용 (NFR-4 LAN 한정 운영, Security Baseline extension 비활성).
- **Compression (permessage-deflate)**: 메시지 크기 작아 효과 미미, PoC 단순성 우선.
- **Subprotocol negotiation**: protocolVersion 정보용 필드만 (Q-FD-U3-13=B), 협상 없음.
- **Rate limiting / DOS 방어**: 신뢰된 LAN 클라이언트 가정. 64 KiB read limit으로 단순 가드만.
- **Fuzz testing**: PoC 범위 외 (Q-NFR-U3-12=A).
