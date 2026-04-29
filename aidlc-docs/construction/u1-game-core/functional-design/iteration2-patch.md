# U1 Game Core — Functional Design Iteration 2 Patch

**문서 버전**: 1.0
**작성일**: 2026-04-29
**기준 산출물**: `domain-entities.md`, `business-logic-model.md`, `business-rules.md` (모두 v1, 2026-04-26)
**상위 변경 명세**: `requirements-iteration2-patch.md` v2.0-patch + `application-design/iteration2-patch.md` v1.0
**처리 방식**: v1 본문 보존, 본 patch가 변경분만 정의.

---

## 1. 변경 요약

| ID | 종류 | 위치 | 변경 |
|---|---|---|---|
| **D-1** | 변경 | `Options` (`types.go`) | `MaxPlayers int` 필드 추가 (FR-11.2) |
| **D-2** | 신규 | `Action` (`action.go`) | `EndSelfIntro{ PlayerID PlayerID }` 액션 (FR-12.2) |
| **D-3** | 신규 | `Apply` 핸들러 | `handleEndSelfIntro` (라운드 로빈 advance) |
| **D-4** | 보존 (wire 비노출) | `AdvanceIntro` | 호스트 강제 advance 액션은 코드 보존, U3 wire에서는 비노출 (Out of Scope OOS-4 회피용 호스트 우회 수단으로 사용 안 함) |
| **D-5** | 불변 | `EndReason`, `IntroSpeakerChanged`, `PhaseChanged` | 본 반복은 신규 이벤트 없이 기존 이벤트로 자기소개 자동 진행 표현 |

---

## 2. 변경 디테일

### 2.1 D-1 — Options 확장

```go
type Options struct {
    MafiaCount            int  `json:"mafiaCount"`
    MaxPlayers            int  `json:"maxPlayers"`            // ===== Iteration 2 신규 =====
    IntroSecondsPerPlayer int  `json:"introSecondsPerPlayer"`
    DiscussionSeconds     int  `json:"discussionSeconds"`
    DoctorSelfHealAllowed bool `json:"doctorSelfHealAllowed"`
    AnnouncementVoiceOn   bool `json:"announcementVoiceOn"`
}
```

- `MaxPlayers` 는 호스트가 게임 설정에서 지정. 6 ≤ MaxPlayers ≤ 12 (FR-1.3 / FR-11.2 / CR1-Q4=A).
- v1 의 `validateOptions` 는 본 패치에서 `MaxPlayers` 범위 검증을 추가. 단, **`Engine.Start` 의 인원 검증은 이미 6~12명이며 `len(players) <= opts.MaxPlayers` 추가 검증을 도입**해, 호스트 설정 인원 초과 시 시작을 거부.
- backward compat: 기존 v1 호출 경로(SessionManager Engine.Start) 가 MaxPlayers=0 으로 호출하면 **Engine은 MaxPlayers=0 을 "제한 없음"으로 해석** (validateOptions 내 분기). 즉 v1 테스트는 영향을 받지 않음.

**validateOptions 변경 (의사 코드)**:
```go
func validateOptions(opts Options, n int) error {
    if opts.MaxPlayers != 0 {
        if opts.MaxPlayers < 6 || opts.MaxPlayers > 12 {
            return errf(CodeValidation, "maxPlayers must be 6..12; got %d", opts.MaxPlayers)
        }
        if n > opts.MaxPlayers {
            return errf(CodeValidation, "actual players %d > maxPlayers %d", n, opts.MaxPlayers)
        }
    }
    // 기존 MafiaCount 검증 그대로
    if opts.MafiaCount < 1 {
        return errf(CodeValidation, "mafiaCount must be >= 1")
    }
    if opts.MafiaCount > n - 3 {  // 의사 1, 경찰 1, 시민 ≥ 1
        return errf(CodeValidation, "mafiaCount %d leaves no room for doctor/police/citizens", opts.MafiaCount)
    }
    return nil
}
```

### 2.2 D-2 — EndSelfIntro 액션

```go
// EndSelfIntro is the player-initiated trigger to advance the intro
// round-robin. Allowed only when Phase == INTRO and PlayerID equals the
// current speaker (Players[IntroSpeakerIdx].ID). FR-12.
type EndSelfIntro struct {
    sealedAction
    PlayerID PlayerID
}
```

### 2.3 D-3 — handleEndSelfIntro

```go
func (e *engine) handleEndSelfIntro(a EndSelfIntro) (State, []EventEnvelope, error) {
    if err := ensurePhase(&e.state, PhaseIntro); err != nil {
        return e.state.Clone(), nil, err
    }
    if e.state.IntroSpeakerIdx < 0 || e.state.IntroSpeakerIdx >= len(e.state.Players) {
        return e.state.Clone(), nil, errf(CodeValidation, "intro speaker index out of range")
    }
    current := e.state.Players[e.state.IntroSpeakerIdx].ID
    if a.PlayerID != current {
        return e.state.Clone(), nil, errf(CodePermissionDenied, "EndSelfIntro: %q is not the current speaker (%q)", a.PlayerID, current)
    }
    now := e.clock.Now()
    if e.state.IntroSpeakerIdx < len(e.state.Players)-1 {
        e.state.IntroSpeakerIdx++
        e.state.IntroSpeakerStartedAt = now
        return e.state.Clone(), []EventEnvelope{pub(IntroSpeakerChanged{
            PlayerID:    e.state.Players[e.state.IntroSpeakerIdx].ID,
            SecondsLeft: e.state.Settings.IntroSecondsPerPlayer,
        })}, nil
    }
    return e.transitionIntroToNight(now)
}
```

- 행동: 현재 발언자가 본인 호출이면 다음 발언자로 advance. 마지막 발언자라면 `transitionIntroToNight` 으로 NIGHT 전환.
- 에러: 비-INTRO 단계 → `CodeWrongPhase`. 비-현재발언자 호출 → `CodePermissionDenied`.

### 2.4 D-4 — AdvanceIntro 처리

- **코드 보존** — 기존 `AdvanceIntro` 액션과 `handleAdvanceIntro` 핸들러는 변경하지 않음.
- **wire 미노출** — U3에서 `AdvanceIntro` 액션을 wire 명령으로 받지 않도록 매핑 제거 (FR-12.3 / Iteration 2 OOS-4).
- 본 반복의 자기소개 진행은 `EndSelfIntro` 가 단독 경로.

### 2.5 D-5 — 이벤트 변동 없음

- 기존 `IntroSpeakerChanged` 와 `PhaseChanged` 이벤트로 본인 종료 → 다음 발언자 / NIGHT 전환을 충분히 표현 가능.
- 신규 도메인 이벤트 도입 없음 (U2 의 라이프사이클 이벤트 `RoomOpened` 등은 U2에서 정의).

---

## 3. 단위 테스트 변경 / 추가

### 3.1 신규 테스트

| 테스트 | 위치 | 검증 내용 |
|---|---|---|
| `TestEndSelfIntro_AdvancesToNextSpeaker` | `internal/game/handlers_lifecycle_test.go` | INTRO Phase, idx=0 발언자 본인 종료 → idx=1 발언자로 advance, `IntroSpeakerChanged` 이벤트 발행 |
| `TestEndSelfIntro_LastSpeakerTransitionsToNight` | 동일 | 마지막 발언자 본인 종료 → Phase=NIGHT, `PhaseChanged{NIGHT}` 발행 |
| `TestEndSelfIntro_RejectsNonCurrentSpeaker` | 동일 | idx=0 인 상태에서 idx=1 플레이어가 EndSelfIntro 호출 → `CodePermissionDenied` |
| `TestEndSelfIntro_RejectsInNonIntroPhase` | 동일 | NIGHT 단계에서 EndSelfIntro 호출 → `CodeWrongPhase` |
| `TestOptions_ValidatesMaxPlayers` | `internal/game/validation_test.go` | MaxPlayers=5 (하한 미만) / 13 (상한 초과) → 검증 실패. 0 → 미설정으로 통과 |
| `TestOptions_RejectsActualPlayersExceedingMax` | 동일 | n=8, MaxPlayers=6 → 검증 실패 |

### 3.2 회귀 테스트 영향

- v1 의 모든 기존 테스트는 `Options.MaxPlayers` 미설정(=0) 으로 동작하므로 영향 없음.
- `TestEngineStartGame*` 류 — Options에 신규 필드가 추가되어도 기본값 0이라 통과.

---

## 4. 커버리지 목표

- v1 U1 커버리지 90.4% 유지.
- 신규 핸들러 (`handleEndSelfIntro`) 가 4개 테스트로 전 분기 커버: advance / transition / non-current / wrong-phase.
- 신규 검증 (`validateOptions` MaxPlayers 분기) 2개 분기 커버.

---

## 5. 다음 단위 의존 (U2 인터페이스)

본 patch가 U2에 노출하는 신규 surface:
1. `game.EndSelfIntro` 액션 타입
2. `game.Options.MaxPlayers` 필드

U2 SessionManager 는 이를 wire `player:end-self-intro` 메시지에서 변환하여 `Engine.Apply` 에 전달, MaxPlayers는 `host:create-room` 페이로드에서 읽어 Options에 채움.
