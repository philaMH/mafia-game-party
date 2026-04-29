# Unit Test Instructions — mafia-game

**작성일**: 2026-04-26
**대상**: U1~U5 단위 테스트 실행 + 커버리지 측정

본 문서는 단위별 단위 테스트(`*_test.go`, `*.test.{ts,tsx}`)의 실행 절차와 통과 기준을 정의합니다.

---

## 1. 단위 테스트 실행 — 전체

### 1.1 Go 백엔드 (U1~U4)

```bash
go test ./internal/... ./cmd/mafia-game/...
```

**기대 결과**:
```
ok  github.com/saltware/mafia-game/internal/game
ok  github.com/saltware/mafia-game/internal/announce
ok  github.com/saltware/mafia-game/internal/persistence
ok  github.com/saltware/mafia-game/internal/session
ok  github.com/saltware/mafia-game/internal/transport/ws
ok  github.com/saltware/mafia-game/internal/transport/http
```

### 1.2 Race detector

```bash
go test -race ./internal/... ./cmd/mafia-game/...
```

NFR-U2-C2 / NFR-U3-C2 / NFR-U4-C2 — 모두 통과 필수.

### 1.3 React SPA (U5)

```bash
cd web && npm test
```

**기대 결과**:
```
Test Files  4 passed (4)
     Tests  32 passed (32)
```

---

## 2. 단위별 상세 실행

### 2.1 U1 Game Core

```bash
go test ./internal/game/
go test -coverprofile=/tmp/u1-cover.out ./internal/game/
go tool cover -func=/tmp/u1-cover.out | tail -1
```

**파일 인벤토리**: 16 테스트 파일 (apply / end / error / handlers_* / property / scenario / tally / tick / role / keyword / types / validation / fixtures).

**통과 기준**:
- 모든 테스트 통과
- 라인 커버리지 ≥ **85%** (NFR-U1-M3, 실측 90.4%)

### 2.2 U2 Session, Persistence & Announce

```bash
go test ./internal/session/... ./internal/announce/... ./internal/persistence/...
go test -coverprofile=/tmp/u2-cover.out \
  ./internal/session/... ./internal/announce/... ./internal/persistence/...
go tool cover -func=/tmp/u2-cover.out | tail -1
```

**파일 인벤토리**: persistence 5 + announce 1 + session 8 = 14 테스트 파일.

**통과 기준**:
- 모든 테스트 통과 (`-race` 포함)
- 합산 라인 커버리지 ≥ **85%** (NFR-U2-M1, 실측 86.5%)

### 2.3 U3 Realtime Transport

```bash
go test ./internal/transport/ws/...
go test -race ./internal/transport/ws/...
go test -coverprofile=/tmp/u3-cover.out ./internal/transport/ws/...
```

**파일 인벤토리**: 6 테스트 (protocol / client / writer / dispatch / handlers / integration).

**통과 기준**:
- E2E 통합 테스트 (`integration_test.go`) 통과 — net.Pipe 기반
- 라인 커버리지 ≥ **85%** (NFR-U3-M1, 실측 89.0%)
- goroutine 누수 0 (`TestE2E_LeakNoGoroutineGrowth`)

### 2.4 U4 HTTP Bootstrap & Static

```bash
go test ./internal/transport/http/...
go test -coverprofile=/tmp/u4-cover.out ./internal/transport/http/...
```

**파일 인벤토리**: 6 테스트 (server / middleware / routes / api_results / lan / integration).

**통과 기준**:
- /api/results JSON에 `members[].token` 미포함 검증 (NFR-U4-S1)
- graceful shutdown < 5초 (NFR-U4-R1)
- 라인 커버리지 ≥ **85%** (NFR-U4-M1, 실측 89.2%)

### 2.5 U5 Web Frontend

```bash
cd web
npm run typecheck
npm run lint
npm test
npm run test:coverage
```

**파일 인벤토리**: 5 테스트 (setup + reducer + useToken + useTTSQueue + NicknameForm).

**통과 기준**:
- 모든 vitest 테스트 통과 (32/32)
- 핵심 모듈 라인 커버리지 ≥ **70%** (NFR-U5-M3, 실측 78.72%)

---

## 3. 통합 검증 명령 (CI 한 번에)

```bash
#!/usr/bin/env bash
set -euo pipefail

echo "=== Go: vet/fmt ==="
go vet ./...
test -z "$(gofmt -l ./internal/ ./cmd/)"

echo "=== Go: test -race ==="
go test -race ./internal/... ./cmd/mafia-game/...

echo "=== Go: coverage ==="
go test -coverprofile=/tmp/all.out \
  ./internal/game/... \
  ./internal/announce/... \
  ./internal/persistence/... \
  ./internal/session/... \
  ./internal/transport/ws/... \
  ./internal/transport/http/...
go tool cover -func=/tmp/all.out | tail -1

echo "=== TS: typecheck/lint/test ==="
( cd web && npm run typecheck && npm run lint && npm test )

echo "=== Build single binary ==="
( cd web && npm run build )
go build -o mafia-game ./cmd/mafia-game

echo "=== ALL UNIT TESTS PASSED ==="
```

---

## 4. 통과 기준 매트릭스

| 단위 | 테스트 통과 | -race | 커버리지 목표 | 실측 |
|---|:---:|:---:|---:|---:|
| U1 | ✅ 16 파일 | ✅ | ≥ 85% | **90.4%** |
| U2 | ✅ 14 파일 | ✅ | ≥ 85% | **86.5%** |
| U3 | ✅ 6 파일 | ✅ | ≥ 85% | **89.0%** |
| U4 | ✅ 6 파일 | ✅ | ≥ 85% | **89.2%** |
| U5 | ✅ 4 파일 (32 cases) | N/A (single thread) | ≥ 70% (core) | **78.72%** |

---

## 5. 흔한 실패 시나리오

| 증상 | 원인 | 해결 |
|---|---|---|
| `go test` race detector flake | 일시적 timing | `go test -race -count=3` 재실행 |
| `vitest` jsdom warning | 미정 환경 변수 | `tests/setup.ts` 점검 |
| coverage 미달 | 신규 코드에 테스트 부재 | 어떤 함수가 0%인지 `go tool cover` 확인 |
| persistence 테스트 SQLite 권한 에러 | tempdir 정리 실패 | `t.TempDir()` 사용 확인 |

---

## 6. 검증 체크리스트

- [x] 단위별 명령 5개 (U1~U5)
- [x] 통합 CI 스크립트
- [x] 통과 기준 매트릭스 (커버리지 목표 vs 실측)
- [x] race detector 게이트 (NFR-U2/U3/U4-C2)
- [x] U5 typecheck + lint 명시
- [x] 실패 시나리오 표
