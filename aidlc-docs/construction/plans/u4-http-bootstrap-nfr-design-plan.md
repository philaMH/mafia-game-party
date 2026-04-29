# NFR Design Plan — U4 HTTP Bootstrap & Static

**작성일**: 2026-04-26
**대상 단위**: U4 / HTTP Bootstrap & Static (`cmd/mafia-game`, `internal/transport/http`)
**참조**:
- `nfr-requirements.md` (NFR-U4-R/P/M/S/C/B)
- `tech-stack-decisions.md`
- `functional-design/*.md`

> 본 plan은 U4 NFR Design의 단일 진실 소스입니다.

---

## 0. NFR Design의 목적

NFR Requirements에서 정한 한도를 만족시키기 위한 패턴(타임아웃, 미들웨어, 종료 시퀀스, embed 처리)과 논리 컴포넌트 정의.

---

## 1. 단계 체크리스트

- [x] (1) NFR Req → 패턴 매핑
- [x] (2) 결정 질문 작성 (Q-NFRD-U4-1~7)
- [x] (3) plan 문서 작성 (본 파일)
- [x] (4) 사용자 답변 수집 — "승인" (권장 답안)
- [x] (5) 답변 일관성 검증
- [x] (6) `nfr-design-patterns.md` 작성
- [x] (7) `logical-components.md` 작성
- [x] (8) audit + aidlc-state 갱신, 사용자 승인 게이트

---

## 2. 결정 질문 (Q-NFRD-U4-1 ~ Q-NFRD-U4-7)

### Q-NFRD-U4-1. http.Server 타임아웃 설정

`http.Server`의 ReadHeaderTimeout / ReadTimeout / WriteTimeout / IdleTimeout 값?

- **A.** **ReadHeaderTimeout=10s, ReadTimeout=30s, WriteTimeout=0(WS 호환), IdleTimeout=60s**. WS 업그레이드는 WriteTimeout=0이 필수.
- **B.** 모두 30초.
- **C.** 모두 무제한.

[Answer]: A

### Q-NFRD-U4-2. logging middleware 패턴

요청 로깅 구현?

- **A.** **statusRecorder 래퍼 + slog.Info(method, path, status, duration_ms)** — 응답 본문 미기록.
- **B.** httputil.NewSingleHostReverseProxy처럼 외부 lib.
- **C.** 로깅 안 함.

[Answer]: A

### Q-NFRD-U4-3. 정적 자산 핸들러 — 캐시 헤더 설정 위치

immutable 캐시 헤더는 어디서 설정?

- **A.** **assetsHandler 미들웨어 layer** — http.FileServerFS 호출 전 `w.Header().Set`.
- **B.** Vite 빌드 시 별도 메타데이터.
- **C.** 클라이언트 service worker.

[Answer]: A

### Q-NFRD-U4-4. signal 처리 — context vs select

shutdown 시그널 받은 후 종료 흐름?

- **A.** **`signal.NotifyContext(ctx, SIGINT, SIGTERM)` 사용 — ctx 취소가 시그널 = ctx.Done()으로 통합**.
- **B.** select 채널.
- **C.** atomic flag polling.

[Answer]: A

### Q-NFRD-U4-5. embed.FS placeholder 처리

`web/dist`가 비어있을 때 빌드 동작?

- **A.** **`//go:embed all:web/dist`이 빈 디렉터리는 허용. placeholder index.html을 git에 commit해 항상 1 파일 보장**.
- **B.** 빈 디렉터리 시 컴파일 에러.
- **C.** 동적 fallback HTML.

[Answer]: A

### Q-NFRD-U4-6. /api/results 응답 인코딩

JSON encoder 선택?

- **A.** **`json.NewEncoder(w).Encode(resp)` — 직접 stream**. 메모리 적게 사용.
- **B.** json.Marshal → w.Write.
- **C.** 별도 라이브러리.

[Answer]: A

### Q-NFRD-U4-7. 단위 테스트 패턴

핸들러 단위 테스트는?

- **A.** **httptest.NewServer + http.Get 클라이언트** — 통합 친화. mock SessionManager / Hub / Store는 자체 stub 작성.
- **B.** httptest.ResponseRecorder + 직접 핸들러 호출.
- **C.** A + B 혼합 (간단한 핸들러는 ResponseRecorder, 통합은 NewServer).

[Answer]: C

---

## 3. 산출물 예상

| 파일 | 책임 |
|---|---|
| `nfr-design-patterns.md` | P-U4-1~7 패턴 (타임아웃, statusRecorder, immutable cache, NotifyContext, embed placeholder, JSON encoder, 테스트 패턴) + 안티패턴 |
| `logical-components.md` | LC-U4-1~N 카탈로그 + 패키지 파일 레이아웃 + NFR Req ↔ LC 매트릭스 |

---

## 4. 사용자 승인 게이트

본 plan과 답변을 검토해 주세요. 모든 답변에 동의하시면 **"완료"** 또는 **"승인"** 으로 응답해 주세요.
