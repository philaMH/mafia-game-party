# Unit of Work — Mafia Game

**작성일**: 2026-04-26
**문서 버전**: 1.0
**참조**: `requirements.md` v1.1, `application-design/components.md`, `application-design/component-methods.md`, `application-design/services.md`, `plans/unit-of-work-plan.md`

본 문서는 시스템을 5개의 **개발 단위(Unit of Work)** 로 분할하고 각 단위의 책임·구성요소·코드 위치·외부 의존·인터페이스 요약·빌드 산출물을 정의합니다.

> **운영 단위 vs 개발 단위**: 본 시스템은 **단일 Go 바이너리(1개 운영 배포 단위)** 입니다. 본 문서의 U1~U5는 **개발 단위(논리적 모듈)** 이며, 모두 같은 바이너리에 패키지/번들로 통합됩니다.

---

## 0. 코드 조직 전략 (Greenfield)

### 0.1 모노레포 단일 모듈 구조

```
mafia-game/                              # 워크스페이스 루트 (Greenfield)
├── go.mod                               # 단일 Go 모듈 (Q-UG-3=A)
├── go.sum
├── cmd/
│   └── mafia-game/
│       └── main.go                      # U4: 부트스트랩
├── internal/
│   ├── game/                            # U1: Game Core (도메인)
│   │   ├── engine.go
│   │   ├── role.go
│   │   ├── keyword.go
│   │   ├── statemachine.go
│   │   ├── types.go                     # 공용 도메인 타입 (Q-UG-5=A 단일 정의처)
│   │   └── *_test.go
│   ├── session/                         # U2: Session
│   │   ├── manager.go
│   │   └── *_test.go
│   ├── announce/                        # U2: AnnouncementService
│   │   ├── announcer.go
│   │   ├── catalog.go
│   │   └── *_test.go
│   ├── persistence/                     # U2: PersistenceStore (SQLite)
│   │   ├── sqlite_store.go
│   │   ├── schema.go
│   │   └── *_test.go
│   └── transport/
│       ├── ws/                          # U3: Realtime Transport
│       │   ├── hub.go
│       │   ├── client.go
│       │   ├── protocol.go              # 와이어 메시지 정의
│       │   └── *_test.go
│       └── http/                        # U4: HTTP Bootstrap
│           ├── router.go
│           ├── embed.go                 # //go:embed web/dist
│           └── lan.go
├── web/                                 # U5: Web Frontend (React+Vite)
│   ├── package.json
│   ├── vite.config.ts
│   ├── tsconfig.json
│   ├── index.html
│   ├── src/
│   │   ├── main.tsx
│   │   ├── App.tsx
│   │   ├── routes.tsx                   # /public, /play 라우트
│   │   ├── views/
│   │   │   ├── PublicView/              # C8 PublicView
│   │   │   └── PlayerView/              # C9 PlayerView
│   │   ├── ws/
│   │   │   └── client.ts                # WebSocket 클라이언트
│   │   ├── tts/
│   │   │   └── queue.ts                 # Web Speech API 큐잉/폴백
│   │   ├── types/                       # 백엔드 와이어 타입과 공유 (수동 동기화 또는 codegen)
│   │   └── components/                  # 공통 UI
│   └── dist/                            # 빌드 산출물 (Go가 embed) — gitignore 권장
└── data/                                # 런타임 SQLite (자동 생성, gitignore)
```

### 0.2 빌드 파이프라인

1. `cd web && npm run build` → `web/dist/` 생성
2. `go build ./cmd/mafia-game` → `mafia-game` 단일 바이너리 (web/dist 동봉, NFR-7)
3. 실행 → `data/mafia.db` 자동 생성

---

## 1. U1 — Game Core (Domain)

| 항목 | 내용 |
|---|---|
| **ID / 이름** | U1 / Game Core |
| **계층** | 도메인 (Domain) |
| **목적** | 마피아 게임 한 판의 **상태 머신과 비즈니스 규칙** 보유. 외부 I/O 0. |
| **포함 컴포넌트** | C1 GameEngine, C2 RoleAssigner |
| **코드 위치** | `internal/game/*` |
| **외부 의존** | 표준 라이브러리만 (`math/rand`, `time` 등) |
| **인터페이스 요약** | `GameEngine.Start/Apply/Tick/Snapshot/Restore`, `RoleAssigner.Assign` (자세한 내용은 `component-methods.md`) |
| **공용 타입 정의처** | **Yes** — `PlayerID`, `Role`, `Phase`, `Action`, `Event`, `State`, `Assignments` 등 도메인 타입의 단일 정의처 (Q-UG-5=A) |
| **빌드 산출물** | Go 패키지 (단일 바이너리에 링크) |
| **개발 순서** | **1순위** (Bottom-up, Q-UG-4=A) |

### 책임 상세
- 단계 전이 (Lobby → Day1.Intro → Night → Day → Vote → … → End)
- 역할 배분 (인원수 기반) 및 키워드 부여
- 마피아 살해 / 의사 보호 / 경찰 조사 적용
- 투표 집계, 동률 처리(재투표 1회 → 무처형)
- 종료 조건 판정 (마피아 0 / 마피아 ≥ 시민)
- 자기소개 시간 진행, 토론 타이머 (Tick 입력)
- 스냅샷 직렬화/복원

### 비책임
- 영속화, WebSocket, HTTP, UI, 안내 메시지(한국어 문자열) — 다른 단위에서 처리

---

## 2. U2 — Session, Persistence & Announcement

| 항목 | 내용 |
|---|---|
| **ID / 이름** | U2 / Session & Persistence |
| **계층** | 애플리케이션 + 인프라 |
| **목적** | **단일 GM 락**으로 게임 세션을 직렬 처리하고, 스냅샷을 디스크에 영속화하며, 도메인 이벤트를 한국어 안내 메시지(자막+TTS 텍스트)로 변환. |
| **포함 컴포넌트** | C3 SessionManager, **C4 AnnouncementService**, C5 PersistenceStore |
| **코드 위치** | `internal/session/*`, `internal/announce/*`, `internal/persistence/*` |
| **외부 의존** | `modernc.org/sqlite` (순수 Go SQLite) |
| **인터페이스 요약** | `SessionManager.CreateSession/JoinPlayer/StartGame/SubmitAction/HostControl/Tick`, `AnnouncementService.Render`, `PersistenceStore.SaveSnapshot/LoadActiveSnapshot/SaveResult/ListResults` |
| **빌드 산출물** | Go 패키지 (단일 바이너리에 링크) + 런타임 `data/mafia.db` |
| **개발 순서** | **2순위** |
| **상위 의존** | U1 (도메인 타입·이벤트) |

### 책임 상세
- 세션 라이프사이클 (1세션 동시 운영, FR-1.1)
- 플레이어 입장·재연결 (닉네임 식별, FR-1.2)
- GameEngine 호출 → 결과 이벤트 후처리
- **단계 전이마다 동기 스냅샷** (NFR-1 안정성)
- 호스트 권한 식별 및 통제 명령 게이팅 (Q-AD-6=B)
- 도메인 이벤트 → 한국어 자막 + TTS payload 매핑 (FR-8.4 카탈로그)
- 토론 잔여 시간 안내 등 시간 기반 트리거 (`Tick` 위임)
- 게임 결과 누적 저장 (FR-6.1) 및 조회 (FR-6.3)

### 비책임
- WebSocket 입출력, 라우팅, 클라이언트 식별 — U3 위임
- 음성 합성(SpeechSynthesis 호출) — 클라이언트(U5)가 텍스트 받아 발화

---

## 3. U3 — Realtime Transport (WebSocket Hub)

| 항목 | 내용 |
|---|---|
| **ID / 이름** | U3 / Realtime Transport |
| **계층** | 인프라 |
| **목적** | 다중 클라이언트 WebSocket 연결을 관리하고, 단위 U2와 UI 단위 U5 사이의 양방향 메시지 라우팅. |
| **포함 컴포넌트** | C6 WSHub |
| **코드 위치** | `internal/transport/ws/*` |
| **외부 의존** | `github.com/gorilla/websocket` (Q-AD-2=A) |
| **인터페이스 요약** | `WSHub.Register/Unregister/Dispatch/OnAction` (자세한 내용은 `component-dependency.md` §와이어 포맷) |
| **빌드 산출물** | Go 패키지 (단일 바이너리에 링크) |
| **개발 순서** | **3순위** |
| **상위 의존** | U2 (SessionManager 진입점), U1 (이벤트 직렬화) |

### 책임 상세
- 클라이언트 연결/해제, ping/pong, 자동 재연결 후보 처리
- 클라이언트 식별 (PlayerID 또는 PublicViewer)
- 도메인 이벤트 → 대상별 메시지 직렬화/송신
  - 공용: 모든 PublicView + 살아있는 모든 PlayerView
  - 비공개: 특정 PlayerID 또는 역할군(예: 마피아 야간 채널)
- 클라이언트 입력(Action) 수신 → SessionManager 위임
- 와이어 프로토콜 정의 (`internal/transport/ws/protocol.go`)

### 비책임
- HTTP 라우팅·정적 자산 — U4 (단, `/ws` 업그레이드는 U4가 호출)
- 비즈니스 규칙 — U1·U2

---

## 4. U4 — HTTP Bootstrap & Static Assets

| 항목 | 내용 |
|---|---|
| **ID / 이름** | U4 / HTTP Bootstrap & Static |
| **계층** | 인프라 / 부트스트랩 |
| **목적** | 단일 바이너리 진입점. HTTP 라우팅, React SPA 정적 자산 동봉 서빙, WebSocket 업그레이드, 호스트 LAN IP 콘솔 출력. |
| **포함 컴포넌트** | C7 HTTPServer + `cmd/mafia-game/main.go` |
| **코드 위치** | `cmd/mafia-game/main.go`, `internal/transport/http/*` |
| **외부 의존** | `net/http`, `embed` (표준 라이브러리) |
| **인터페이스 요약** | `HTTPServer.NewRouter/PrintLANAddresses` |
| **빌드 산출물** | **단일 Go 바이너리** (`mafia-game`) — `web/dist`를 `//go:embed` |
| **개발 순서** | **4순위** |
| **상위 의존** | U2, U3 (와이어링 대상), U5 (정적 자산 빌드 산출물) |

### 책임 상세
- `GET /`, `GET /public`, `GET /play` → React SPA 진입 (history fallback)
- `GET /ws` → WebSocket 업그레이드 → WSHub 위임
- `GET /api/results` (선택) → `PersistenceStore.ListResults` (FR-6.3)
- 시작 시 콘솔에 `http://<LAN-IP>:<port>` 출력 (FR-1.1 보조)
- 부팅 시 모든 단위 와이어링 (의존성 주입)

### 비책임
- 프론트엔드 빌드 (별도 단계: `npm run build` 후 `go build`)
- 비즈니스 로직, WebSocket 라우팅 본체

---

## 5. U5 — Web Frontend (React SPA)

| 항목 | 내용 |
|---|---|
| **ID / 이름** | U5 / Web Frontend |
| **계층** | 프레젠테이션 |
| **목적** | 한 React SPA에서 `/public`(공용 화면 + TTS + 자막)과 `/play`(개인 화면) 두 라우트를 모두 제공. |
| **포함 컴포넌트** | C8 PublicView, C9 PlayerView |
| **코드 위치** | `web/src/*` |
| **기술 스택** | React + Vite + TypeScript (Q-AD-3=C+사용자 명시), Web Speech API |
| **외부 의존** | `react`, `react-router-dom` (또는 동등), Vite 빌드 도구. 음성: 브라우저 Web Speech API |
| **인터페이스 요약** | WebSocket 와이어 프로토콜의 클라이언트 측 (U3 정의 참조) + Web Speech 큐잉 |
| **빌드 산출물** | `web/dist/` (정적 자산) → U4가 `embed.FS`로 동봉 |
| **개발 순서** | **5순위** (마지막) |
| **상위 의존** | U3 (와이어 프로토콜), U2 (이벤트 의미) |

### 책임 상세 (PublicView, `/public`)
- 단계/타이머/사망자/투표 결과/종료 화면 렌더링
- **TTS 큐잉/인터럽션** (FR-8.6) — `SpeechSynthesisQueue`, 단계 전환 시 인터럽트
- 음성 ON/OFF 토글 (FR-8.5)
- Web Speech API 가용성 점검 → 부재 시 토스트 + 자막 폴백 (시나리오 6)
- 호스트 컨트롤 패널 (게임 시작/일시정지/토론 조기 종료/강제 종료)

### 책임 상세 (PlayerView, `/play`)
- 닉네임 입력/입장
- 자기 역할·키워드 비공개 표시 (FR-2.3, FR-3.2) — **음성 출력 X** (FR-8.2)
- 단계별 입력 UI (마피아 살해 / 의사 보호 / 경찰 조사 / 투표)
- 마피아 대표자 입력 (Q-AD-7) — 마지막 입력 채택, 다른 마피아는 현재 선택 상태 확인
- 재연결 시 자기 상태 자동 복원

### 비책임
- 백엔드 비즈니스 규칙·영속화 — 모든 결정은 U1~U2가 보유, 클라이언트는 표시·입력만

---

## 6. 단위 요약 매트릭스

| Unit | 단위명 | 계층 | 외부 의존 | 빌드 산출물 | 순서 |
|---|---|---|---|---|---|
| U1 | Game Core | 도메인 | 표준 lib | Go 패키지 | 1 |
| U2 | Session & Persistence (+Announce) | 애플리케이션+인프라 | `modernc.org/sqlite` | Go 패키지 + `data/mafia.db` | 2 |
| U3 | Realtime Transport | 인프라 | `gorilla/websocket` | Go 패키지 | 3 |
| U4 | HTTP Bootstrap & Static | 인프라/부트스트랩 | `net/http`, `embed` | **단일 바이너리 `mafia-game`** | 4 |
| U5 | Web Frontend | 프레젠테이션 | React, Vite, Web Speech API | `web/dist/` (U4가 동봉) | 5 |

---

## 7. 단위 경계·검증 체크리스트

- [x] 단위 간 의존이 단방향(상위 → 하위만 의존)이며 순환 없음 (자세한 내용은 `unit-of-work-dependency.md`)
- [x] 도메인(U1)은 외부 I/O·프레임워크에 비의존
- [x] 모든 컴포넌트(C1~C9)가 정확히 1개 단위에 할당됨
- [x] 모든 FR(FR-1~FR-8)이 단위 책임에 매핑됨 (자세한 내용은 `unit-of-work-story-map.md`)
- [x] 모든 NFR(특히 NFR-1 안정성)이 단위 책임에 매핑됨
- [x] 단일 운영 산출물(단일 바이너리) 보장 (NFR-7)
- [x] 공용 도메인 타입의 단일 정의처(U1) 명시
