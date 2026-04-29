# Domain Entities — U3 Realtime Transport

**작성일**: 2026-04-26
**문서 버전**: 1.0
**참조**: `requirements.md` v1.1 FR-1.1/FR-1.2 + NFR-2, `application-design/component-methods.md` C6, `u2-public-api.md`, `u1-public-api.md`, `plans/u3-realtime-transport-functional-design-plan.md`

본 문서는 U3의 도메인 엔티티(클라이언트 추상, 와이어 메시지)와 Hub 인터페이스를 정의합니다. 비즈니스 규칙(`business-rules.md`)과 흐름(`business-logic-model.md`)은 자매 문서를 참조.

---

## 1. ClientKind / ClientID (Q-FD-U3-1=A)

```
ClientKind = "PUBLIC" | "PLAYER"

ClientID = string  // 서버 발급 무작위 ID (PlayerID와 별개)
```

| 값 | 의미 | PlayerID 보유 |
|---|---|:---:|
| `PUBLIC` | 공용 화면 (`/public`, TTS + 자막) | ✗ |
| `PLAYER` | 개인 화면 (`/play`, 토큰 인증 후) | ✓ |

> 본 PoC는 `Host`를 별도 ClientKind로 두지 않습니다 — 호스트는 `PLAYER` 중 `PlayerID == HostID`인 자입니다. 권한 체크는 SessionManager가 담당 (Q-FD-U3-12=B).

---

## 2. Client (Hub 내부 상태)

```
Client {
    ID              ClientID
    Kind            ClientKind
    PlayerID        game.PlayerID    // PUBLIC이면 빈 값
    Conn            *websocket.Conn  // gorilla/websocket
    Out             chan []byte       // 송신 채널, 버퍼 16 (Q-FD-U3-7=A)
    JoinedAt        time.Time
    LastPongAt      time.Time         // 하트비트 갱신
    Closed          bool              // double-close 방지
}
```

### 불변식

- 같은 `PlayerID`를 가진 `PLAYER` Client는 동시에 1개만 활성. 신규 연결 시 기존 연결 강제 종료 (Q-FD-U3-9=A, last-connect-wins).
- `PUBLIC` Client는 PlayerID 없음. 같은 PC에 여러 Public 탭이 열려도 모두 활성.
- `Out` 채널이 가득 차면 해당 Client만 disconnect — Hub의 다른 클라이언트는 영향 없음.

---

## 3. Hub 인터페이스 (LC 후속 단계 확정용 요약)

```go
type Hub interface {
    // Register는 이미 업그레이드된 *websocket.Conn을 받아 새 Client를 등록합니다.
    // PUBLIC은 PlayerID="" 로 호출. PLAYER는 별도 join/resume 메시지 처리 후
    // Hub가 내부적으로 PlayerID를 할당합니다 (이 메서드 호출은 Upgrade 직후 1회).
    Register(conn *websocket.Conn) (ClientID, error)

    // Unregister는 Client를 정리하고 WS connection을 닫습니다 (idempotent).
    Unregister(id ClientID)

    // Run은 Hub의 백그라운드 워커(브로드캐스트 dispatch)를 실행합니다.
    // SessionManager.Subscribe 핸들러로 등록된 OnEvent가 외부에서 호출됩니다.
    Run(ctx context.Context) error

    // Close는 모든 Client를 정리하고 Run을 종료시킵니다.
    Close() error
}

func New(
    upgrader websocket.Upgrader,
    mgr session.SessionManager,
    log *slog.Logger,
) Hub
```

> Hub의 외부 진입점은 (1) `Register` (HTTP 핸들러가 호출), (2) `Run`/`Close` (라이프사이클), 두 종류뿐. SessionManager.Subscribe는 Hub 생성자가 내부에서 1회 호출 — 본 단위가 와이어링 책임을 짊어집니다.

---

## 4. ClientRegistry (내부 컴포넌트)

```
ClientRegistry {
    mu              sync.RWMutex
    byID            map[ClientID]*Client
    byPlayerID      map[game.PlayerID]*Client  // PLAYER만 (PUBLIC은 unindexed)
    publics         []*Client
}
```

**책임**:
- 등록/해제 시 byID + byPlayerID + publics 동기화
- 가시성에 따른 대상 클라이언트 스냅샷 반환 (락 보호 하의 read-only iteration)
- last-connect-wins 정책 — 새 Player 등록 시 기존 동일 PlayerID Client를 Unregister 후 등록

---

## 5. 와이어 프로토콜 — 봉투 구조 (Q-FD-U3-3=A)

JSON 평탄 구조: 모든 메시지의 첫 키는 `type` (필수). 추가 필드는 type별로 정의.

```json
{ "type": "<message-type>", "...": "..." }
```

JSON tag 결정성을 위해 모든 송신 메시지는 `omitempty` 옵션을 의도적으로 사용 안 함 (수신측 구현 간소화).

---

## 6. Client → Server 메시지 (incoming)

| `type` | 페이로드 | 설명 | 매핑 SessionManager |
|---|---|---|---|
| `host:create-session` | `{name: string}` | PUBLIC 또는 첫 PLAYER 핸드셰이크에서 호스트가 세션 생성 | `CreateSession` |
| `join` | `{name: string}` | 신규 PLAYER 입장 (LOBBY 한정) | `JoinPlayer` |
| `resume` | `{token: string}` | 재연결 (단계 무관) | `ResumePlayer` |
| `host:start` | `{options: GameOptions}` | 호스트가 게임 시작 | `StartGame` |
| `submit:advance-intro` | `{}` | 호스트 자기소개 단계 진행 | `SubmitAction(AdvanceIntro)` |
| `submit:mafia-kill` | `{target: PlayerID}` | 마피아 대표자 살해 입력 | `SubmitAction(SubmitMafiaKill)` |
| `submit:doctor-heal` | `{target: PlayerID}` | 의사 보호 입력 | `SubmitAction(SubmitDoctorHeal)` |
| `submit:police-check` | `{target: PlayerID}` | 경찰 조사 입력 | `SubmitAction(SubmitPoliceCheck)` |
| `submit:end-night` | `{}` | 호스트 야간 마감 | `SubmitAction(EndNightEarly)` |
| `submit:end-discussion` | `{}` | 호스트 토론 조기 종료 | `SubmitAction(EndDiscussionEarly)` |
| `submit:vote` | `{target: PlayerID}` | 투표 | `SubmitAction(SubmitVote)` |
| `host:toggle-voice` | `{on: bool}` | 음성 안내 토글 | `SubmitAction(ToggleVoice)` |
| `host:force-end` | `{}` | 호스트 강제 종료 | `SubmitAction(ForceEndGame)` |
| `subscribe-public` | `{}` | PUBLIC 클라이언트가 dispatch 대상으로 등록 (인증 불필요) | (Hub 내부) |

> 호스트 권한 체크는 SessionManager가 단독 수행 (Q-FD-U3-12=B). Hub는 메시지의 `type`을 `Action`으로 매핑하는 역할만.

### 봉투 디코딩 패턴

```go
type incomingEnvelope struct {
    Type string          `json:"type"`
    Raw  json.RawMessage `json:",inline"` // 전체 페이로드 보관, 디코딩은 type별
}
```

실제 디코딩은 `switch env.Type`으로 분기 후 typed struct로 `json.Unmarshal(env.Raw, ...)`.

---

## 7. Server → Client 메시지 (outgoing)

가시성에 따라 라우팅됨. 각 메시지는 `EventOut`(U2의 `Envelope` + `Announcement`)에서 파생.

| `type` | 페이로드 | 발생 시점 |
|---|---|---|
| `welcome` | `{clientId, kind, protocolVersion: "v1"}` | Register 직후 (단순화: protocolVersion은 정보용, 검증 안 함 — Q-FD-U3-13=B) |
| `joined` | `{playerId, token, isHost}` | join/resume 성공 응답 |
| `snapshot` | `{state, your: {role, keyword, team, mafiaCohort}, isHost}` | resume 직후 자기 화면 복원 (Q-FD-U3-15=A) |
| `event` | `{event: <typedEvent>, visibility: "PUBLIC"\|"PLAYER"\|"ROLE_MAFIA"}` | 도메인 이벤트 push |
| `announce` | `{subtitle, speech, severity, urgent: bool}` | 한국어 안내 (PUBLIC 전용 — Q-FD-U3-6=A) |
| `error` | `{code, message}` | 입력 거부 시 송신자에게만 |
| `pong` | `{}` | 클라이언트 ping에 대한 응답 (보조 — 표준 WS ping/pong 우선) |

### `event` 페이로드 — 이벤트 타입별 wire 직렬화 규칙

내부 `game.Event`(sealed interface)를 wire JSON으로 변환. 각 typed event는 `kind` 필드 + 자체 페이로드:

```json
{
  "type": "event",
  "visibility": "PUBLIC",
  "event": {
    "kind": "PhaseChanged",
    "phase": "DAY",
    "day": 2,
    "deadline": 1714000000000
  }
}
```

매핑 표 (15개 event type):

| Engine event | wire kind | wire 필드 |
|---|---|---|
| `GameStarted` | `"GameStarted"` | (state는 별도 snapshot으로 보냄, event payload는 비움) |
| `PhaseChanged` | `"PhaseChanged"` | phase, day, deadline (epoch ms) |
| `RoleRevealedToPlayer` | `"RoleRevealedToPlayer"` | role, keyword |
| `MafiaCohortRevealed` | `"MafiaCohortRevealed"` | mafiaIds[], representativeId |
| `IntroSpeakerChanged` | `"IntroSpeakerChanged"` | playerId, secondsLeft |
| `MafiaTargetSelected` | `"MafiaTargetSelected"` | representativeId, target |
| `PoliceResult` | `"PoliceResult"` | police, target, team |
| `DeathAnnounced` | `"DeathAnnounced"` | victim |
| `PeacefulNight` | `"PeacefulNight"` | (없음) |
| `DiscussionTimerTick` | `"DiscussionTimerTick"` | secondsLeft |
| `VoteTallied` | `"VoteTallied"` | counts (map), eliminated?, recount |
| `Eliminated` | `"Eliminated"` | playerId, role |
| `MafiaRepresentativeReassigned` | `"MafiaRepresentativeReassigned"` | oldId, newId |
| `GameEnded` | `"GameEnded"` | winner?, endReason, reveal[] |
| `VoiceToggled` | `"VoiceToggled"` | on |

> **PhaseChanged의 deadline**은 epoch milliseconds로 직렬화 (브라우저 `Date.now()`와 호환). State.Deadline은 Go time.Time → 밀리초 정수.

---

## 8. PUBLIC vs PLAYER 메시지 매트릭스

| 메시지 | PUBLIC | PLAYER | 비고 |
|---|:---:|:---:|---|
| `welcome` | ✅ | ✅ | 등록 직후 |
| `joined` | ✗ | ✅ | join/resume 응답 |
| `snapshot` | ✗ | ✅ | resume 직후 자기 화면 |
| `event` (VisPublic) | ✅ | ✅ (살아있는 자신) | Q-FD-U3-5=A: 사망 PLAYER도 받음 |
| `event` (VisPlayer) | ✗ | ✅ (대상 1인만) | RoleRevealedToPlayer, PoliceResult |
| `event` (VisRoleMafia) | ✗ | ✅ (살아있는 마피아만) | MafiaCohortRevealed 등 |
| `announce` (ForPublicOnly) | ✅ | ✗ | Q-FD-U3-6=A: Hub 자동 필터 |
| `announce` (private error) | ✗ | ✅ (송신자만) | RenderError 결과 |
| `error` | 송신자에게만 | 송신자에게만 | join/resume/submit 거부 |

---

## 9. 기술 스택 결정 요약

| 영역 | 결정 |
|---|---|
| 라이브러리 | `github.com/gorilla/websocket` (Q-AD-2=A) |
| 메시지 인코딩 | JSON (encoding/json 표준) |
| 핸드셰이크 인증 | WS 첫 메시지 join/resume (Q-FD-U3-2=A) |
| 봉투 구조 | 평탄 `{type, ...}` (Q-FD-U3-3=A) |
| 송신 큐 | 클라이언트당 chan []byte 버퍼 16 (Q-FD-U3-7=A) |
| 하트비트 | 30초 read deadline + 25초 ping (Q-FD-U3-8=A) |
| 동시 접속 | last-connect-wins (Q-FD-U3-9=A) |
| 와이어 버전 | 정보용 "v1" 필드 (검증 안 함, Q-FD-U3-13=B) |
| 로깅 | debug 레벨, type만 — 페이로드 미기록 (Q-FD-U3-14=A) |

---

## 10. 검증 체크리스트

- [x] ClientKind 2종 + Client 불변식 명확
- [x] Hub 인터페이스 4 메서드
- [x] ClientRegistry 책임 정의
- [x] 와이어 봉투 평탄 구조
- [x] incoming 메시지 14종 + SessionManager 매핑
- [x] outgoing 메시지 7 type + event kind 15종 매핑
- [x] PUBLIC/PLAYER 라우팅 매트릭스
- [x] 사망 라우팅 (Q-FD-U3-5=A) 명시
