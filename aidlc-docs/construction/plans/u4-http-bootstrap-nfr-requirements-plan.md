# NFR Requirements Plan — U4 HTTP Bootstrap & Static

**작성일**: 2026-04-26
**대상 단위**: U4 / HTTP Bootstrap & Static (`cmd/mafia-game`, `internal/transport/http`)
**참조**:
- `requirements.md` v1.1 NFR-1/2/4/6/7
- `aidlc-docs/construction/u4-http-bootstrap-static/functional-design/*.md`

> 본 plan은 U4 NFR Requirements의 단일 진실 소스입니다.

---

## 0. NFR 영역 우선순위

U4는 부트스트랩 + 정적 자산 + 얇은 API 어댑터 단위. 비즈니스 로직 0.

| 영역 | 적용 여부 | 비고 |
|---|:---:|---|
| Reliability | **상위** | graceful shutdown, signal 처리 |
| Performance | 중간 | 정적 자산 로컬 LAN, 응답 < 50ms 목표 |
| Maintainability | **상위** | 단위 테스트(httptest), 외부 lib 0 |
| Security | 중간 | NFR-4 비공개(Token 제외), LAN 한정 |
| Concurrency | 중간 | net/http가 자체 처리 — 추가 락 0 |
| Scalability / Availability | **N/A** | 단일 호스트 PC, 단일 게임 |

---

## 1. 단계 체크리스트

- [x] (1) 영역 우선순위 평가
- [x] (2) 결정 질문 작성 (Q-NFR-U4-1~10)
- [x] (3) plan 문서 작성 (본 파일)
- [x] (4) 사용자 답변 수집 — "승인" (권장 답안)
- [x] (5) 답변 일관성 검증 — 모호성 없음
- [x] (6) `nfr-requirements.md` 작성
- [x] (7) `tech-stack-decisions.md` 작성
- [x] (8) audit + aidlc-state 갱신, 사용자 승인 게이트

---

## 2. 결정 질문 (Q-NFR-U4-1 ~ Q-NFR-U4-10)

### Q-NFR-U4-1. 외부 의존성 한정

U4에 추가하는 외부 lib는?

- **A.** **0개** — Go 표준 라이브러리만 (net/http, embed, os, os/signal, flag, net, encoding/json, fmt, log/slog).
- **B.** + chi 또는 gorilla/mux 라우터.
- **C.** + viper / cobra (설정·CLI 도구).

[Answer]: A

### Q-NFR-U4-2. /api/results p99 응답 시간

LAN 환경에서 /api/results limit=50 응답 지연 목표는?

- **A.** **p99 < 50 ms** (SQLite ListResults + JSON 직렬화).
- **B.** p99 < 100 ms (느슨).
- **C.** p99 < 20 ms (엄격).

[Answer]: A

### Q-NFR-U4-3. 정적 자산 응답 시간

`/assets/main-abc.js` 등 정적 파일 첫 응답 지연은?

- **A.** **p99 < 20 ms** (embed.FS in-memory + immutable cache 후 304).
- **B.** p99 < 50 ms.
- **C.** 측정 안 함.

[Answer]: A

### Q-NFR-U4-4. 단위 테스트 커버리지 목표

`internal/transport/http/`의 라인 커버리지 목표는?

- **A.** **≥ 80%** — main.go 부팅 코드는 통합 테스트로 일부 커버.
- **B.** ≥ 85% — U2/U3와 동일 수준.
- **C.** ≥ 70% — 얇은 어댑터.

[Answer]: B

### Q-NFR-U4-5. 정적 분석 / 포맷 게이트

빌드 게이트는?

- **A.** **`go vet` 0 issue + `gofmt -l` empty + `go test -race` 통과** (U2/U3와 동일).
- **B.** + golangci-lint.
- **C.** A보다 느슨.

[Answer]: A

### Q-NFR-U4-6. graceful shutdown 시간 한도

SIGTERM 수신 후 종료 완료까지 한도는?

- **A.** **< 7초** (HTTP 5s + Hub 즉시 + mgr 2s, FD §8).
- **B.** < 10초.
- **C.** 무한 대기.

[Answer]: A

### Q-NFR-U4-7. SPA fallback latency

`/play/some/route` → index.html fallback 응답 지연은?

- **A.** **p99 < 30 ms** (embed 메모리 read + ServeContent).
- **B.** 측정 안 함 (정적 파일).

[Answer]: A

### Q-NFR-U4-8. 동시 HTTP 요청 처리 능력

LAN 환경 동시 요청 처리량 목표는?

- **A.** **동시 100 요청 무문제** (16 클라이언트 × 6 자산 + 약간의 API 호출 정도 — 매우 여유).
- **B.** 동시 1000 요청 (PoC 범위 외).
- **C.** 측정 안 함.

[Answer]: A

### Q-NFR-U4-9. 비공개 정보 라우팅 검증 (NFR-4)

/api/results 응답에 토큰 미포함 검증은?

- **A.** **단위 테스트로 응답 JSON 파싱 → "token" 키 부재 확인**.
- **B.** + 통합 테스트.
- **C.** 코드 리뷰만.

[Answer]: A

### Q-NFR-U4-10. embed 빌드 게이트

`web/dist/index.html` placeholder 부재 시?

- **A.** **빌드 실패 (embed 표준 동작)** + CI에서 placeholder 부재 감지하면 재시도.
- **B.** 런타임 503 fallback.
- **C.** placeholder 없이 동적 생성.

[Answer]: A

---

## 3. 산출물 예상

| 파일 | 책임 |
|---|---|
| `nfr-requirements.md` | 6개 영역 NFR 항목 ID + 측정 가능 한도 + 검증 방법 + 트레이드오프 + FR/NFR 추적성 + 검증 게이트 + 비-요구사항 |
| `tech-stack-decisions.md` | 표준 라이브러리만 + 패키지 레이아웃(`internal/transport/http/*` + `cmd/mafia-game/main.go`) + 의존 그래프 |

---

## 4. 사용자 승인 게이트

본 plan과 답변을 검토해 주세요. 변경이 필요하면 답을 수정해 알려주시고, 모든 답변에 동의하시면 **"완료"** 또는 **"승인"** 으로 응답해 주세요.
