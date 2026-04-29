# Component Dependency — Mafia Game

**작성일**: 2026-04-25

본 문서는 컴포넌트 간 의존 관계, 통신 패턴, 데이터 흐름을 정의합니다.

---

## 1. 의존 매트릭스

행(↓)이 열(→)에 의존:

|              | C1 GameEngine | C2 Assigner | C3 Session | C4 Announce | C5 Persist | C6 WSHub | C7 HTTP | C8 Public | C9 Player |
|---|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|
| **C1 GameEngine** | — | ✅ | — | — | — | — | — | — | — |
| **C2 Assigner** | — | — | — | — | — | — | — | — | — |
| **C3 Session** | ✅ | (via C1) | — | ✅ | ✅ | ✅ | — | — | — |
| **C4 Announce** | (event types) | — | — | — | — | — | — | — | — |
| **C5 Persist** | (state types) | — | — | — | — | — | — | — | — |
| **C6 WSHub** | (event types) | — | — | (announcement type) | — | — | — | — | — |
| **C7 HTTP** | — | — | ✅ | — | ✅ | ✅ | — | — | — |
| **C8 Public** (FE) | (wire types) | — | — | (wire types) | — | (WS protocol) | (HTTP) | — | — |
| **C9 Player** (FE) | (wire types) | — | — | (wire types) | — | (WS protocol) | (HTTP) | — | — |

> 괄호 `(...)`는 **타입 의존**(공유 메시지 스키마)만 있는 약한 의존을 의미.

### 의존 규칙
- **도메인 컴포넌트(C1, C2)** 는 외부 I/O·인프라에 의존하지 않음 → 단위 테스트 용이
- **C3 SessionManager**가 모든 협력자(C1/C4/C5/C6)를 합쳐 오케스트레이션 — 다른 컴포넌트 간 직접 호출 금지(혼선 방지)
- **C7 HTTPServer**는 부팅 와이어링만 담당 — 비즈니스 로직 없음
- **프론트엔드(C8/C9)** 는 백엔드와 **WebSocket wire format + REST(`/api/results`)** 만으로 통신

---

## 2. 통신 패턴

### 2.1 동기 호출 (Go process 내부)
| 호출자 | 호출 대상 | 패턴 | 용도 |
|---|---|---|---|
| C3 SessionManager | C1 GameEngine | 메서드 호출 | 단일 스레드 직렬 처리 (락으로 보호) |
| C3 SessionManager | C4 Announce | 메서드 호출 | 이벤트 → 안내 변환 |
| C3 SessionManager | C5 Persist | 메서드 호출 (`context.Context`) | 단계 전이 시 동기 SaveSnapshot |
| C3 SessionManager | C6 WSHub | 메서드 호출 | Dispatch (논블로킹 큐 내부에서 비동기 송신) |

### 2.2 비동기 (WebSocket)
| 송신자 | 수신자 | 채널 | 메시지 |
|---|---|---|---|
| 백엔드 (C6) | 프론트(C8 Public) | 공용 채널 | PhaseChanged, DeathAnnounced, VoteTallied, GameEnded, IntroSpeakerChanged, DiscussionTimerTick + Announcement(자막+TTS) |
| 백엔드 (C6) | 프론트(C9 Player) | 본인 비공개 채널 | RoleRevealedToPlayer, PoliceResult + 단계별 입력 가능 상태 |
| 프론트(C9) | 백엔드 (C6→C3) | 양방향 | 입장(JoinPlayer), 행동 입력(Submit*), 호스트 컨트롤(StartGame/EarlyVote/Abort) |

### 2.3 시간 기반 진전
- 백엔드의 `time.Ticker`(예: 200ms)가 `SessionManager.Tick(now)` 호출 → `GameEngine.Tick`이 단계 전환을 자율적으로 진행 (자기소개 1인당 N초, 토론 타이머, 야간 마감 등)

### 2.4 단일 GM 락
- `SessionManager`는 단일 `sync.Mutex` 또는 단일 액터 고루틴(`chan` 직렬화)로 동시 입력을 직렬 처리 → 상태 불변성 보장 (NFR-1)

---

## 3. 데이터 흐름 다이어그램 (텍스트)

```
[ Player Browser (C9) ]                            [ Public Browser (C8) ]
        │  WS                                              │  WS + Web Speech API
        ▼                                                  ▼
+-------------------------------- WSHub (C6) -------------------------------+
        │                                                  ▲
        ▼                                                  │ Dispatch
+----- SessionManager (C3) -------------+                  │
|  Apply(action)                        |                  │
|     ├── GameEngine.Apply  (C1)        |                  │
|     │     └── RoleAssigner (C2)*      |                  │
|     ├── Persistence.Save  (C5)        |                  │
|     └── Announce.Render   (C4) ───────┴──────────────────┘
+----------------------------------------+
        │  load on boot
        ▼
+--- PersistenceStore (C5, SQLite) ---+
|  data/mafia.db                      |
+-------------------------------------+

[* RoleAssigner: 게임 시작 시점에만 호출]
```

---

## 4. 와이어 포맷 (잠정 — 코드 단계에서 확정)

### 4.1 클라이언트 → 서버
```jsonc
// 입장
{ "type": "join", "name": "철수" }

// 게임 시작 (호스트 한정)
{ "type": "host:start", "options": { "introSecondsPerPlayer": 30, "discussionSeconds": 90, "doctorSelfHealAllowed": true } }

// 마피아 살해 입력
{ "type": "submit:mafia-kill", "target": "p_07" }

// 호스트 토론 조기 종료
{ "type": "host:early-vote" }
```

### 4.2 서버 → 클라이언트
```jsonc
// 안내 (자막+TTS)
{ "type": "announce", "text": "밤이 깊어졌습니다.", "voice": { "lang": "ko-KR", "pitch": 0.8, "rate": 0.9 }, "urgent": true }

// 단계 변경 (공용)
{ "type": "phase", "phase": "NIGHT", "day": 1, "deadline": 1714000000000 }

// 비공개: 본인 역할 공개
{ "type": "you:role", "role": "POLICE", "keyword": "정의" }
```

> 정확한 타입/필드는 Functional Design 또는 Code Generation 단계에서 코드 우선으로 확정.

---

## 5. 외부 의존성 요약

| 종류 | 라이브러리 | 사용처 | 비고 |
|---|---|---|---|
| WebSocket | `github.com/gorilla/websocket` | C6 WSHub | Q-AD-2=A |
| SQLite | `modernc.org/sqlite` | C5 PersistenceStore | 순수 Go (cgo 불필요), Q-AD-1=A |
| 임베드 | `embed` (표준) | C7 HTTPServer | React 빌드 산출물 동봉 |
| 로깅 | `log/slog` (표준) | 전역 | Go 1.21+ |
| 프론트엔드 | React + Vite + TypeScript | C8 PublicView, C9 PlayerView | Q-AD-3=C(React) |
| 프론트엔드(TTS) | Web Speech API (브라우저 내장) | C8 PublicView | FR-8.1 |

> 외부 클라우드 서비스, 외부 인터넷 의존성 없음 — LAN 한정 운영(NFR-7).
