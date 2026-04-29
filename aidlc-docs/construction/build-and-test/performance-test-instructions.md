# Performance Test Instructions — mafia-game

**작성일**: 2026-04-26
**대상**: NFR-2 / NFR-U2-P / NFR-U3-P / NFR-U4-P / NFR-U5-P 검증

본 문서는 LAN 환경에서 측정 가능한 성능 한도(p99 지연, 동접, 빌드 크기)를 어떻게 확인할지 정의합니다.

---

## 1. 성능 목표 매트릭스

| 단위 | 항목 | 한도 | 측정 방법 |
|---|---|---|---|
| U2 | SaveSnapshot p99 | < 50 ms | Go benchmark + WAL DB |
| U2 | SubmitAction (lock acquire ~ release) p99 | < 100 ms | Go benchmark |
| U2 | LoadActiveSnapshot p99 | < 50 ms | Go benchmark |
| U3 | SubmitAction → 첫 클라이언트 push p99 | < 200 ms | net.Pipe 시나리오 + time.Since |
| U3 | 동시 16 연결 안정 동작 | 모두 성공 | 통합 테스트 |
| U4 | /api/results p99 | < 50 ms | curl + hyperfine |
| U4 | /assets p99 | < 20 ms | curl |
| U4 | 동시 100 HTTP 요청 | 모두 200 | wrk |
| U5 | wire → DOM 갱신 p99 | < 100 ms | DevTools Performance |
| U5 | 빌드 산출물 (gzip) | < 500 KB | vite build 출력 |

---

## 2. U2 — SaveSnapshot / SubmitAction benchmark

### 2.1 측정 코드 (예시)

`internal/persistence/sqlite_store_bench_test.go` (필요 시 작성):

```go
package persistence_test

import (
    "context"
    "encoding/json"
    "path/filepath"
    "testing"
    "time"

    "github.com/saltware/mafia-game/internal/game"
    "github.com/saltware/mafia-game/internal/persistence"
)

func BenchmarkSaveSnapshot(b *testing.B) {
    dir := b.TempDir()
    store, _ := persistence.OpenSqlite(context.Background(), filepath.Join(dir, "bench.db"))
    defer store.Close()

    snap := persistence.Snapshot{ /* 12명 정도 채움 */ }
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = store.SaveSnapshot(context.Background(), snap)
    }
}
```

### 2.2 실행

```bash
go test -bench=BenchmarkSaveSnapshot -benchtime=5s ./internal/persistence/
```

**판정 기준**: `ns/op`을 ms로 환산했을 때 p99 < 50 ms. 평균(`ns/op`)이 < 5ms이면 p99도 통상 < 50 ms.

### 2.3 SubmitAction 전체 지연

```go
func BenchmarkSubmitAction(b *testing.B) {
    mgr, _ := newTestManager(b)
    host, _ := makeLobby(b, mgr, 6)
    mgr.StartGame(ctx, host.PlayerID, game.DefaultOptions(6))
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = mgr.SubmitAction(ctx, game.ToggleVoice{HostID: host.PlayerID, On: true})
    }
}
```

**판정 기준**: ns/op → ms 환산 < 100 ms.

---

## 3. U3 — WS push 지연

### 3.1 net.Pipe 기반 직접 측정

`integration_test.go`의 `TestE2E_HostJoinStartReceivesEvents`를 다음처럼 변경:

```go
start := time.Now()
sendJSON(t, host, map[string]any{"type": "host:start", "options": ...})
readUntil(t, host, func(typ string, _ []byte) bool { return typ == "event" })
t.Logf("StartGame -> first event: %v", time.Since(start))
```

**판정 기준**: 로그 출력의 시간이 < 200 ms.

### 3.2 16 동접 안정성

기존 `TestE2E_LeakNoGoroutineGrowth`를 16개 conn으로 늘려 회귀 테스트로 사용.

---

## 4. U4 — /api/results / /assets 지연

### 4.1 hyperfine (외부 도구)

```bash
./mafia-game --port 18080 &
SERVER=$!
sleep 1

# 미리 결과 데이터 100개 시드
sqlite3 data/mafia.db <<EOF
INSERT INTO game_results (game_id, started_at, ended_at, end_reason, options_json, members_json, reveal_json)
SELECT 'g-' || hex(randomblob(4)), datetime(), datetime(), 'CITIZEN_WIN', '{}', '[]', '[]'
FROM (WITH RECURSIVE c(x) AS (VALUES(1) UNION ALL SELECT x+1 FROM c WHERE x<100) SELECT x FROM c);
EOF

hyperfine --warmup 10 --runs 200 \
  "curl -fsS http://localhost:18080/api/results"

kill $SERVER
```

**판정 기준**: hyperfine 출력의 p99(또는 max)가 < 50 ms.

### 4.2 정적 자산

```bash
hyperfine --warmup 10 --runs 200 \
  "curl -fsS http://localhost:18080/assets/main-{hash}.js > /dev/null"
```

**판정 기준**: max < 20 ms (cache 미사용 첫 응답 기준).

### 4.3 100 동시 요청 (wrk)

```bash
wrk -t4 -c100 -d10s http://localhost:18080/healthz
```

**판정 기준**: 0 errors, latency p99 < 100 ms.

---

## 5. U5 — DOM 갱신 / 빌드 크기

### 5.1 DOM 갱신 지연 (수동)

1. Chrome DevTools → Performance 탭 → Record 시작
2. 호스트 화면에서 "게임 시작" 클릭 (다른 디바이스에서도 시작 → push 트리거)
3. PUBLIC 화면에 GameStarted 자막 표시될 때까지 측정
4. Performance 타임라인에서 "Recalculate Style" + "Layout" + "Paint" 합산

**판정 기준**: 메시지 수신 → paint complete가 < 100 ms.

### 5.2 빌드 산출물 크기

```bash
cd web && npm run build
```

빌드 마지막 라인:
```
../cmd/mafia-game/web/dist/assets/index-{hash}.js   183.96 kB │ gzip: 60.14 kB
```

**판정 기준**: gzip < 500 KB. **실측 60.14 KB ✅**.

### 5.3 Lighthouse FCP

1. `./mafia-game` 실행
2. Chrome → 시크릿 모드 → http://localhost:8080/public
3. DevTools → Lighthouse → "Performance" 카테고리만 → Generate

**판정 기준**: First Contentful Paint < 1초 (LAN, 캐시 미사용).

---

## 6. 회귀 방지 — CI에 추가할 것

| 명령 | 통과 기준 | 빈도 |
|---|---|---|
| `go test -bench=. -benchtime=2s ./internal/persistence/` | ns/op trend | nightly |
| `cd web && npm run build` 후 dist 크기 확인 | gzip < 500 KB | 매 commit |
| `wrk` smoke test | 0 errors | 주간 |

---

## 7. 명시적 비-측정 항목 (Non-Goals)

- 1000 동접 부하 — PoC 범위 외 (NFR-7)
- TLS 핸드셰이크 시간 — 미사용
- DB row 100만 ListResults — 1년 운영 100판 가정
- 모바일 4G 환경 latency — LAN 가정

---

## 8. 검증 체크리스트

- [x] 단위별 성능 목표 매트릭스 (10항목)
- [x] U2 Go benchmark 실행 절차
- [x] U3 net.Pipe 측정 + 16 동접 안정성
- [x] U4 hyperfine + wrk 절차
- [x] U5 DevTools + Lighthouse 절차
- [x] 빌드 산출물 크기 자동 검증 (CI)
- [x] 비-측정 항목 명시
