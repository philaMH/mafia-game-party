# U2 Session/Persistence/Announce · Code Generation Plan — Iteration 8

**Status**: Draft v1.0 — 사용자 승인 대기
**Source**: `aidlc-docs/construction/u2-session-persistence-announce/functional-design/iteration8-patch.md` v1.0 (사용자 승인 2026-04-29T22:15Z)
**Type**: Bug Fix Minimal Patch (catalog 분기 1건)

---

## 1. Step 개요

```
Step A — catalog_default.go: NightStepChanged{INTRO} silent 분기 1건 추가
Step B — catalog_test.go:    I8-A1 (INTRO IsEmpty) 단언 1건 추가
Step C — 검증 + audit/state 동기화
```

---

## 2. Step A — `internal/announce/catalog_default.go`

### A.1 NightStepChanged switch 에 INTRO 케이스
```go
case game.NightStepChanged:
    switch e.Step {
    case game.NightStepIntro:
        // Iteration 8: PhaseChanged{NIGHT} 가 phase.night cue 로 안내 담당.
        return Announcement{}
    case game.NightStepMafia:
        return ann(msgNightStepMafia, cueNightMafia, SeverityEmphasis)
    case game.NightStepPolice:
        return ann(msgNightStepPolice, cueNightPolice, SeverityEmphasis)
    case game.NightStepDoctor:
        return ann(msgNightStepDoctor, cueNightDoctor, SeverityEmphasis)
    }
    return Announcement{}
```

### 체크리스트
- [ ] A.1 INTRO 케이스 추가 + 의도 코멘트

---

## 3. Step B — `internal/announce/catalog_test.go`

### B.1 `TestRender_NightStepChanged` 에 INTRO IsEmpty 단언 추가

```go
intro := render(t, game.NightStepChanged{Step: game.NightStepIntro}, game.VisPublic)
if !intro.IsEmpty() {
    t.Errorf("INTRO step should be silent, got %+v", intro)
}
```
- 기존 `RESOLVED` 와 동일 패턴 (매트릭스 외부에 두 단언이 나란히 위치).

### 체크리스트
- [ ] B.1 I8-A1 단언 추가

---

## 4. Step C — 검증 + 동기화

- [ ] `go vet ./internal/announce/...` PASS
- [ ] `go test ./internal/announce/... -count=1 -race` PASS
- [ ] `go test ./... -count=1` 6 패키지 PASS
- [ ] `go test ./internal/announce -coverprofile=/tmp/iter8-announce.out` → 커버리지 ≥ 94.0% 유지
- [ ] audit.md 갱신, aidlc-state.md U2 섹션 [x]

---

## 5. 영향 받는 파일

| 파일 | 라인 변동 |
|---|---|
| `internal/announce/catalog_default.go` | +5 |
| `internal/announce/catalog_test.go` | +5 |
| **합계** | **+10** |

---

## 6. 변경 이력

| 버전 | 일자 | 변경 |
|---|---|---|
| v1.0 | 2026-04-29 | 최초 작성 |
