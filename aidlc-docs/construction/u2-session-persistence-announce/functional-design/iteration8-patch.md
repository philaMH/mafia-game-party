# U2 Session/Persistence/Announce · Functional Design Patch — Iteration 8

**Status**: Draft v1.0 — 사용자 승인 대기
**Source**: `aidlc-docs/inception/requirements/iteration8-fix-vote-result-requirements.md` v1.0
**Plan**: `aidlc-docs/construction/plans/iteration8-execution-plan.md` v1.0
**Predecessor**: Iteration 4 (NightStepChanged 안내 추가), Iteration 7 (Voice mp3 cue)
**Type**: Minimal Patch (catalog 분기 1건)

---

## 1. 변경 요약

`NightStepChanged{NightStepIntro}` 이벤트는 카탈로그에서 빈 `Announcement{}` 를 반환하여 **silent** 처리한다. 안내 음성은 동일 시점에 함께 emit 되는 `PhaseChanged{PhaseNight}` 의 기존 cue (`phase.night` — "밤이 되었습니다…") 가 담당한다 (Q4=B).

---

## 2. 카탈로그 분기 (defaultCatalog.Render)

```go
// internal/announce/catalog_default.go
case game.NightStepChanged:
    switch e.Step {
    case game.NightStepIntro:
        // Iteration 8: INTRO 안내는 동일 시점의 PhaseChanged{NIGHT} 가
        // `phase.night` cue 로 담당. 별도 cue 발화하지 않음.
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

- 신규 mp3 cue 상수 / `msgNightStep*` 메시지 상수 추가 없음 (Q4=B 정책).
- `NightStepResolved` 의 기존 silent 처리는 default branch (`return Announcement{}`) 가 유지.

---

## 3. 테스트 변경

### 3.1 `catalog_test.go::TestRender_NightStepChanged` 매트릭스 확장
- 기존 케이스: `MAFIA`, `POLICE`, `DOCTOR` (subtitle 비어있지 않음을 검증).
- 신규: `NightStepIntro` 케이스를 매트릭스 외부로 분리 — `IsEmpty()` 검증 (RESOLVED 와 동일 형식).

### 3.2 신규 단언 (I8-A1)

```go
intro := render(t, game.NightStepChanged{Step: game.NightStepIntro}, game.VisPublic)
if !intro.IsEmpty() {
    t.Errorf("INTRO step should be silent, got %+v", intro)
}
```

### 3.3 카탈로그 매트릭스 (`TestRender_PhaseChangedAllPhases` 등) 영향 없음
- `PhaseChanged{NIGHT}` 의 `phase.night` cue 발화는 변경 없음.

---

## 4. 영향 받는 파일

| 파일 | 변경 |
|---|---|
| `internal/announce/catalog_default.go` | `NightStepIntro` 케이스 1건 추가 (3 라인) |
| `internal/announce/catalog_test.go` | I8-A1 검증 1건 (5~7 라인) |

---

## 5. 회귀 영향

- `phase.night` cue 흐름 변동 없음 — INTRO 단계 진입 시 기존 발화 그대로.
- 클라이언트(U5) 의 자막 표시 — `phase.night` 자막이 INTRO 5초 + MAFIA 시작 후 `night.mafia` 자막으로 교체. 사용자 체감은 "밤이 되었습니다 → (5초) → 마피아의 시간입니다" 순.
- 호스트 mp3 큐는 FIFO 직렬 재생 — `phase.night.mp3` 종료 후 큐 비어있다가 5초 뒤 `night.mafia.mp3` 가 큐잉/재생.

---

## 6. 변경 이력

| 버전 | 일자 | 변경 |
|---|---|---|
| v1.0 | 2026-04-29 | 최초 작성 |
