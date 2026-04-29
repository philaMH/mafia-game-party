# Components — Mafia Game

**작성일**: 2026-04-25
**참조**: `requirements.md` v1.1, `application-design-plan.md`, `execution-plan.md`

본 문서는 시스템을 구성하는 컴포넌트와 각각의 책임, 인터페이스를 정의합니다.
**상세 비즈니스 로직은 Functional Design 단계에서 결정됩니다.**

---

## 0. 패키지/디렉터리 레이아웃 (잠정)

```
mafia-game/
├── cmd/
│   └── mafia-game/
│       └── main.go                 # HTTPServer 부트스트랩
├── internal/
│   ├── game/                       # 도메인 (순수 Go, 외부 의존 없음)
│   │   ├── engine.go               # GameEngine
│   │   ├── role.go                 # Role, RoleAssigner
│   │   ├── keyword.go              # 키워드 풀
│   │   └── statemachine.go         # 단계 전이
│   ├── session/                    # 애플리케이션 계층
│   │   └── manager.go              # SessionManager
│   ├── announce/
│   │   └── announcer.go            # AnnouncementService (FR-8 카탈로그 매핑)
│   ├── persistence/
│   │   └── sqlite_store.go         # PersistenceStore (SQLite 구현)
│   └── transport/
│       ├── ws/
│       │   └── hub.go              # WSHub (gorilla/websocket)
│       └── http/
│           └── router.go           # HTTPServer
├── web/                            # React SPA (Vite)
│   ├── src/
│   │   ├── views/
│   │   │   ├── PublicView.tsx      # /public 라우트
│   │   │   └── PlayerView.tsx      # /play 라우트
│   │   ├── tts/                    # Web Speech API 큐잉/인터럽션
│   │   └── ws/                     # WebSocket 클라이언트
│   └── dist/                       # 빌드 산출물 (Go embed.FS 대상)
└── data/                           # 런타임 SQLite 파일 (자동 생성)
```

> Go 빌드 시 `web/dist/`를 `embed.FS`로 동봉하여 단일 바이너리(NFR-7)를 산출.

---

## 1. 도메인 컴포넌트

### C1. GameEngine
- **위치**: `internal/game`
- **목적**: 마피아 게임 한 판의 **상태 머신과 비즈니스 규칙**을 책임진다. 외부 I/O 없음 (순수 도메인).
- **주요 책임**:
  - 단계 전이 관리 (Lobby → Day1.Intro → Night → Day → Vote → … → End)
  - 역할 행동 적용 (마피아 살해 / 의사 보호 / 경찰 조사)
  - 투표 집계 + 동률 처리(재투표 1회 → 무처형)
  - 종료 조건 판정 (마피아 0 / 마피아 ≥ 시민)
  - 자기소개 키워드 부여 (RoleAssigner 활용)
- **인터페이스 (개념)**:
  ```go
  type GameEngine interface {
      Start(players []PlayerID, opts Options) (State, []Event, error)
      Apply(action Action) (State, []Event, error)
      Tick(now time.Time) (State, []Event, error)
      Snapshot() State
      Restore(s State) error
  }
  ```

### C2. RoleAssigner
- **위치**: `internal/game`
- **목적**: 인원수 기반 역할 배분 + 매 게임 무작위 키워드 부여 (FR-2.2, FR-3.1).
- **주요 책임**:
  - 6–12명 인원 테이블에 따라 마피아·의사·경찰·시민 분배
  - 역할별 키워드 풀에서 무작위 1개 추출 (동일 역할은 동일 키워드 부여)
- **인터페이스 (개념)**:
  ```go
  type RoleAssigner interface {
      Assign(playerIDs []PlayerID, seed int64) (Assignments, error)
  }
  ```

---

## 2. 애플리케이션 컴포넌트

### C3. SessionManager
- **위치**: `internal/session`
- **목적**: **단일 활성 게임 세션**의 라이프사이클을 관리하고, 단일 GM 락을 보장한다 (NFR-1).
- **주요 책임**:
  - 게임방 생성/해체 (1세션 동시 운영 제한, FR-1.1)
  - 플레이어 입장·재연결 처리 (닉네임 식별, FR-1.2)
  - GameEngine 호출 → 결과 이벤트를 WSHub에 전달
  - PersistenceStore에 상태 영속화 (단계 전이마다 스냅샷)
  - 호스트 권한(Q-AD-6=B) 식별 및 통제 명령(시작/일시정지/강제종료/조기 토론 종료) 게이팅

### C4. AnnouncementService
- **위치**: `internal/announce`
- **목적**: GameEngine 이벤트를 **사용자 친화 안내 메시지(자막+TTS 텍스트)로 변환**한다 (FR-8.4 카탈로그 매핑).
- **주요 책임**:
  - 이벤트 → 한국어 안내 문자열 매핑 (예: `NightEntered` → "밤이 깊어졌습니다…")
  - TTS payload(텍스트, 큐잉 속성) 생성
  - 토론 잔여 30초 등 시간 기반 안내 트리거

---

## 3. 인프라 컴포넌트

### C5. PersistenceStore
- **위치**: `internal/persistence`
- **선택**: **SQLite** (Q-AD-1=A) — `modernc.org/sqlite` (순수 Go) 권장
- **목적**: 게임 상태 스냅샷·이벤트 로그·결과 요약을 디스크에 영속화 (NFR-1, FR-6).
- **주요 책임**:
  - 진행 중 게임 상태 스냅샷 저장/조회 (서버 재시작 시 복원)
  - 종료된 게임 결과 요약 누적 저장 (FR-6.1)
  - 단순 조회(과거 결과 목록)
- **인터페이스 (개념)**:
  ```go
  type PersistenceStore interface {
      SaveSnapshot(ctx context.Context, s State) error
      LoadActiveSnapshot(ctx context.Context) (State, bool, error)
      SaveResult(ctx context.Context, r GameResult) error
      ListResults(ctx context.Context, limit int) ([]GameResult, error)
  }
  ```

### C6. WSHub
- **위치**: `internal/transport/ws`
- **선택**: **`github.com/gorilla/websocket`** (Q-AD-2=A)
- **목적**: 클라이언트 WebSocket 연결을 다중 관리하고, 이벤트를 라우팅한다 (NFR-2 동기화).
- **주요 책임**:
  - 클라이언트 연결/해제, 핑/퐁, 자동 재연결 지원 (NFR-1)
  - 클라이언트 식별 (PlayerID 또는 PublicViewer)
  - 도메인 이벤트 → 대상별 메시지 직렬화 + 송신
    - 공용 이벤트: 모든 PublicView + 살아있는 모든 PlayerView
    - 비공개 이벤트: 특정 PlayerID 또는 역할군만 (예: 마피아 야간 채널)
  - 클라이언트 입력(Action) 수신 → SessionManager로 위임

### C7. HTTPServer
- **위치**: `internal/transport/http` + `cmd/mafia-game/main.go`
- **목적**: HTTP 라우팅, 정적 자산 서빙, 호스트 IP 노출.
- **주요 책임**:
  - `GET /` → React SPA 진입 (embed.FS)
  - `GET /public` / `GET /play` → SPA 라우트 처리 (history fallback)
  - `GET /ws` → WebSocket 업그레이드 → WSHub 위임
  - `GET /api/results` (선택) → 과거 결과 목록 (FR-6.3)
  - 시작 시 콘솔에 `http://<LAN-IP>:<port>` 출력 (FR-1.1 보조)

---

## 4. 프레젠테이션 컴포넌트 (React SPA)

### C8. PublicView (`/public`)
- **위치**: `web/src/views/PublicView.tsx`
- **목적**: **공용 화면** — 모두가 함께 보는 진행 정보 + **TTS 음성 안내** + 자막 (FR-8.2).
- **주요 책임**:
  - WebSocket으로 공용 이벤트 수신 → 단계/타이머/사망자/투표 결과/종료 화면 렌더링
  - **TTS 큐잉/인터럽션** (FR-8.6): 도착 메시지를 SpeechSynthesisQueue에 push, 큰 단계 전환 시 인터럽트
  - 음성 ON/OFF 토글 (FR-8.5)
  - Web Speech API 가용성 자동 점검 → 부재 시 토스트 + 자막 폴백 (시나리오 6)
  - 호스트 컨트롤 패널 (게임 시작, 일시정지, 토론 조기 종료, 강제 종료)

### C9. PlayerView (`/play`)
- **위치**: `web/src/views/PlayerView.tsx`
- **목적**: **개인 화면** — 자기 비공개 정보(역할·키워드)와 비공개 입력(밤 행동·투표)을 다룸.
- **주요 책임**:
  - 닉네임 입력/입장
  - 자기 역할·키워드 비공개 표시 (FR-2.3, FR-3.2) — **음성은 출력하지 않음** (FR-8.2)
  - 단계별 입력 UI (마피아 살해 / 의사 보호 / 경찰 조사 / 투표)
  - **마피아 대표자 입력 정책** (Q-AD-7): 마피아 중 한 명만 입력 가능 — 마지막 입력이 채택, 다른 마피아는 현재 선택 상태 확인 가능 (정확한 규칙은 Functional Design 확정)
  - 재연결 시 자기 상태 자동 복원 (시나리오 2)

---

## 5. 컴포넌트 매트릭스 요약

| ID | 이름 | 계층 | 외부 의존 | 핵심 책임 한 줄 |
|---|---|---|---|---|
| C1 | GameEngine | 도메인 | 없음 | 상태 머신·규칙 |
| C2 | RoleAssigner | 도메인 | 없음 | 역할/키워드 배분 |
| C3 | SessionManager | 애플리케이션 | C1, C5, C6, C4 | 세션 라이프사이클·락 |
| C4 | AnnouncementService | 애플리케이션 | (이벤트 입력) | 안내 메시지 생성 |
| C5 | PersistenceStore | 인프라 | SQLite(`modernc.org/sqlite`) | 상태/결과 영속화 |
| C6 | WSHub | 인프라 | `gorilla/websocket` | WebSocket 라우팅 |
| C7 | HTTPServer | 인프라 | `net/http`, `embed.FS` | HTTP·정적자산·업그레이드 |
| C8 | PublicView | 프레젠테이션(React) | Web Speech API | 공용 화면·TTS·자막 |
| C9 | PlayerView | 프레젠테이션(React) | — | 개인 화면·비공개 입력 |
