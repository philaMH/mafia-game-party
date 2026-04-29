# Services — Mafia Game

**작성일**: 2026-04-25

본 문서는 컴포넌트들이 협력해 비즈니스 흐름을 수행하는 **서비스 레이어**의 정의와 오케스트레이션을 다룹니다. 본 시스템은 작은 단일 바이너리 도구이므로 명시적 서비스는 두 가지로 단순화합니다.

---

## S1. SessionService (= SessionManager)

### 책임
- 단일 게임 세션의 라이프사이클 오케스트레이션 (생성·진행·종료·정리)
- **GameEngine** (도메인 규칙)과 **PersistenceStore** (영속화), **WSHub** (전송), **AnnouncementService** (안내) 사이를 잇는 **퍼사드(facade)**
- 단일 GM 락 보장 (NFR-1, 동시 호스트 제어 차단)
- 호스트 권한 명령 게이팅

### 의존
- GameEngine (C1)
- RoleAssigner (C2, GameEngine 내부)
- AnnouncementService (C4)
- PersistenceStore (C5)
- WSHub (C6)
- 시계(Clock) — 테스트 가능성을 위해 주입

### 오케스트레이션 패턴

#### Pattern A — 외부 입력 처리
```
WSHub.OnAction(action)
  └─ SessionService.SubmitAction(action)
       ├─ GameEngine.Apply(action) → (newState, events)
       ├─ PersistenceStore.SaveSnapshot(newState)        ← NFR-1 (단계 전이 시 영속화)
       ├─ AnnouncementService.Render(events)             ← FR-8 자막+TTS 텍스트 생성
       └─ WSHub.Dispatch(announcements) + DispatchEvent(privateEvents)
```

#### Pattern B — 시간 기반 진전 (백그라운드 ticker)
```
ticker.Tick(every 200ms)
  └─ SessionService.Tick(now)
       ├─ GameEngine.Tick(now) → (newState, events)
       ├─ if events not empty: SaveSnapshot + Render + Dispatch
       └─ (no events면 no-op — 멱등)
```

#### Pattern C — 호스트 컨트롤
```
WSHub host-only message  →  SessionService.HostControl(hostID, cmd)
  ├─ verify hostID == state.HostID (단일 GM 락)
  ├─ map cmd → game.Action (예: HostEarlyVote → EndDiscussionEarly)
  └─ run Pattern A
```

#### Pattern D — 부팅 시 복원
```
main()
  ├─ store := persistence.NewSQLite("data/mafia.db")
  ├─ snapshot, found := store.LoadActiveSnapshot()
  ├─ engine := game.New(...)
  ├─ if found: engine.Restore(snapshot)
  └─ session := session.New(engine, store, hub, announce.New())
```

### 트랜잭션·일관성 정책
- `Apply` 또는 `Tick`이 새 상태를 반환한 경우, **반드시 `SaveSnapshot` 성공 후에 `Dispatch`** (디스크가 진실의 원천 — NFR-1 데이터 무결성)
- `SaveSnapshot` 실패 시 새 상태를 외부에 노출하지 않음 (사용자에게는 에러 토스트)

---

## S2. AnnouncementService

### 책임
- 도메인 이벤트를 **사용자 친화 안내** (자막 텍스트 + TTS 페이로드)로 변환
- FR-8.4 안내 메시지 카탈로그를 단일 진실 원천으로 보유
- 인터럽트(`urgent=true`) 정책 적용: 단계 전환·게임 종료·동률 발표 등 큰 변화 시

### 입력 → 출력 매핑 (예시)
| 도메인 이벤트 | 안내 (자막+TTS) | Public | Urgent |
|---|---|---|---|
| `GameStarted` | "마피아 게임이 시작됩니다…" | ✅ | ✅ |
| `PhaseChanged{Phase=INTRO, Day=1}` | "첫째날 낮입니다. 차례대로 자기소개를 해주세요." | ✅ | ✅ |
| `IntroSpeakerChanged{Player, SecondsLeft=N}` | "○○○님 차례입니다." | ✅ | ❌ |
| `PhaseChanged{Phase=NIGHT}` | "밤이 깊어졌습니다. 모두 눈을 감으세요." | ✅ | ✅ |
| `PhaseChanged{Phase=NIGHT}` (마피아 차례) | "마피아는 살해할 대상을 지목하세요." | ✅ | ❌ |
| `RoleRevealedToPlayer` | "당신의 역할은 ○○○입니다. 키워드: ○○○" | ❌ (비공개) | — |
| `PoliceResult{Team=...}` | "조사 대상은 ○○○입니다." | ❌ (비공개, 경찰에게만) | — |
| `DeathAnnounced` | "지난밤 ○○○님이 사망했습니다." | ✅ | ✅ |
| `PeacefulNight` | "지난밤은 평화로웠습니다." | ✅ | ✅ |
| `DiscussionTimerTick{SecondsLeft=30}` | "토론 종료까지 30초 남았습니다." | ✅ | ❌ |
| `VoteTallied{Eliminated}` | "투표 결과 ○○○님이 처형되었습니다." | ✅ | ✅ |
| `VoteTallied{Recount=true}` | "투표가 동률입니다. 재투표를 진행합니다." | ✅ | ✅ |
| `GameEnded{Winner}` | "게임이 종료되었습니다. ○○팀의 승리입니다." | ✅ | ✅ |

### 톤 정책
- 모든 음성: `ko-KR`, `pitch=0.8`, `rate=0.9` (근엄, FR-8.3)
- 비공개 이벤트(`Public=false`)는 자막만, TTS 발화 없음 (FR-8.2)

### 확장성 (FR-7)
- 안내 카탈로그를 외부 파일(JSON/YAML)로 분리 가능. 한국어만 우선, 다국어는 향후 확장.

---

## 서비스-컴포넌트 관계 (요약)

```
┌─────────────────────────────────────────────────────────────────┐
│                       SessionService (S1)                       │
│   ┌─────────┐   ┌──────────────┐   ┌──────┐   ┌──────────────┐  │
│   │ Engine  │──▶│ AnnouncementSv│──▶│ WSHub│──▶ Public/Player │  │
│   │  (C1)   │   │     (S2/C4)   │   │ (C6) │   │  Views (C8/9)│  │
│   └────┬────┘   └──────────────┘   └──────┘   └──────────────┘  │
│        │                                                        │
│        ▼                                                        │
│   ┌──────────────┐                                              │
│   │ Persistence  │ ← SaveSnapshot, LoadActiveSnapshot           │
│   │   (C5)       │                                              │
│   └──────────────┘                                              │
└─────────────────────────────────────────────────────────────────┘
                             ▲
                             │
                       HTTPServer (C7) — /ws upgrade, static assets
```
