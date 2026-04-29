# Functional Design Plan — U2 Session, Persistence & Announce

**작성일**: 2026-04-26
**대상 단위**: U2 (`internal/session/*`, `internal/announce/*`, `internal/persistence/*`)
**참조**:
- `requirements.md` v1.1 (FR-1.1, FR-1.2, FR-4.3, FR-6, FR-8.4, NFR-1)
- `application-design/components.md` C3·C4·C5
- `application-design/component-methods.md`
- `application-design/services.md`
- `application-design/unit-of-work.md` §2
- `application-design/unit-of-work-story-map.md` §4 U2 Primary
- `construction/u1-game-core/code/u1-public-api.md` (U1 의존)

---

## 0. U2 단위 컨텍스트 요약

**책임**: 단일 GM 락으로 게임 세션을 직렬 처리, SQLite 스냅샷·결과 영속화, 도메인 이벤트를 한국어 안내 메시지(자막+TTS 텍스트)로 변환.

**Primary 요구사항**:
- FR-1.1 (단일 호스트 PC 1세션, LAN URL 노출 보조)
- FR-1.2 (닉네임 기반 식별 + 재연결)
- FR-4.3 (마피아 대표자 입력 — 권한 게이팅은 U2가 수행)
- FR-6.1 (결과 누적 저장), FR-6.2 (진행 중 상태 스냅샷), FR-6.3 (과거 결과 조회)
- FR-7.2 (안내 카탈로그 외부화)
- FR-8.4 (풍부한 안내 카탈로그)
- NFR-1 (스냅샷 동기 저장, 단일 GM 락, 재연결 시 자기 화면 복원)
- NFR-7 (외부 서비스 0 — SQLite 파일 1개)

**의존**:
- U1 Game Core (도메인 타입, Engine 인터페이스 — 직접 import)
- 외부 lib: `modernc.org/sqlite` (순수 Go SQLite 드라이버) — `database/sql` 표준 인터페이스로 사용

**미확정 디테일** (Application Design §9 + 본 단계 신규):
1. 단일 GM 락 구현 형태 (sync.Mutex 단순 잠금 vs actor 채널)
2. AnnouncementService 카탈로그 데이터 모델 + 외부화 형태
3. 스냅샷 저장 시점·정책 (모든 단계 전이? Apply마다? 비동기 큐?)
4. 호스트 재시작 후 재개 정책 (자동 복원 vs 사용자 확인 후 복원)
5. 재연결 식별 정책 (PlayerID 발급 시점, 토큰/쿠키 vs 닉네임)
6. Tick 실행 주체 (SessionManager가 백그라운드 ticker 보유 vs 외부에서 호출)
7. 에러 → 안내 메시지 매핑 (개발자 에러 코드 → 한국어 사용자 안내 위치)
8. SQLite 스키마 (테이블 분할: snapshots/results/audit, 인덱스, 파일 위치)
9. 다중 게임 이력 누적 (한 번에 진행 중인 게임은 1개, 완료된 결과는 누적)
10. 저장된 비공개 정보 보호 (Role/Keyword를 그대로 BLOB에 저장 vs 마스킹 옵션)
11. 안내 카탈로그 한국어 톤 (근엄한 진행자 톤, CR2 free-form 합치)

---

## 1. 작업 체크리스트 (계획)

- [x] (1) 단위 컨텍스트 분석
- [x] (2) 결정 질문지 작성 (§3)
- [x] (3) plan에 [Answer]: 임베드
- [x] (4) 사용자 답변 수집
- [x] (5) 모순/모호성 점검 + 후속 질문
- [x] (6) 사용자 승인

## 2. 작업 체크리스트 (생성, 승인 후)

- [x] (7) `domain-entities.md` — 세션·스냅샷·결과·안내 카탈로그 데이터 모델
- [x] (8) `business-logic-model.md` — SessionManager 흐름(생성/입장/액션/Tick/스냅샷/복원/재연결), AnnouncementService 매핑, PersistenceStore 트랜잭션
- [x] (9) `business-rules.md` — 권한 게이팅, 락 정책, 영속화 규칙, 안내 카탈로그 결정 표
- [x] (10) frontend-components.md 미작성 (U2 백엔드 단위, 정당화 기록)
- [x] (11) FR/NFR 추적성 검증
- [x] (12) `aidlc-state.md` 갱신
- [x] (13) 사용자 승인 게이트

---

## 3. 결정 질문 (사용자 답변 필요)

> 답변: `[Answer]: <기호>` 또는 자유 의견. 모두 권장이라면 "권장안"으로 답해주셔도 됩니다.

---

### Q-FD-U2-1. 단일 GM 락 구현 형태 (NFR-1)

- A. **`sync.Mutex` 단순 잠금** — SessionManager의 모든 공개 메서드를 `s.mu.Lock(); defer s.mu.Unlock()`으로 감쌈. 단순·이해 쉬움 (권장)
- B. **단일 액터 + 채널 큐** — SessionManager가 고루틴 1개로 모든 요청을 직렬 처리. 더 명시적이지만 코드 복잡↑
- C. Other (자유의견)

[Answer]: A 

---

### Q-FD-U2-2. 스냅샷 저장 시점 (NFR-1)

- A. **모든 PhaseChanged + 모든 사망 이벤트 직후 동기 저장** — Apply 반환 후 즉시 SaveSnapshot. 단순, 안전 (권장)
- B. **모든 Apply 직후** — 빈도 높음, 디스크 I/O 부담 가능
- C. **PhaseChanged만** — 야간 행동 누적 중 손실 가능성 (마피아 입력 등)
- D. **비동기 워커 큐** — 빠르지만 손실 위험 (NFR-1 위협)
- E. Other

[Answer]: A

---

### Q-FD-U2-3. 호스트 재시작 후 자동 복원 정책 (시나리오 3)

- A. **자동 복원** — 바이너리 시작 시 active 스냅샷 발견하면 자동 Restore. 호스트 화면에 "복원됨" 토스트만 표시 (권장 — NFR-1 안정성 최우선)
- B. **호스트 확인 후 복원** — "이전 게임을 이어서 시작할까요? Y/N" UI
- C. **무조건 새 게임으로 시작, 이전 스냅샷 별도 보관** — 재개 X
- D. Other

[Answer]: A

---

### Q-FD-U2-4. 재연결 식별 정책 (FR-1.2, NFR-1)

- A. **세션 토큰** — JoinPlayer 시 SessionManager가 PlayerID + opaque token 발급, 클라이언트 localStorage 저장 → 재연결 시 token 제시 (권장)
- B. **닉네임 기반** — 같은 닉네임으로 재접속 시 같은 PlayerID 매핑. 단순하지만 동명이인 충돌
- C. Other

[Answer]: A

---

### Q-FD-U2-5. Tick 실행 주체

- A. **SessionManager가 백그라운드 ticker 보유** (`time.NewTicker(1 * time.Second)` 고루틴 1개) — 외부 와이어링 단순. 종료 시 graceful stop (권장)
- B. **외부(`cmd/main.go`)가 1초 ticker로 SessionManager.Tick 호출** — Composition Root에서 명시적
- C. Other

[Answer]: A

---

### Q-FD-U2-6. 에러 → 안내 메시지 매핑 위치

U1의 EngineError 9종을 호스트·플레이어 화면에 어떻게 표시할지:

- A. **U2 AnnouncementService가 에러 코드 → 한국어 메시지 매핑** + WS 응답으로 전송. PublicView는 토스트, PlayerView는 폼 옆 에러 표시 (권장)
- B. **U5 프론트엔드가 코드별 한국어 매핑** — 백엔드는 코드만 보냄
- C. **혼합** — 일반 에러는 백엔드, 폼 검증은 프론트
- D. Other

[Answer]: A

---

### Q-FD-U2-7. 안내 카탈로그 데이터 모델

FR-8.4 (풍부한 진행 안내 카탈로그) — 안내 메시지를 어떻게 표현?

- A. **이벤트 타입 → 한국어 템플릿 함수** (Go 코드 내부 — `func renderDeathAnnounced(e DeathAnnounced) string`). 변수 보간은 함수 내. (권장 — 단순, 빌드 시점 검증)
- B. **이벤트 타입 → Go template 문자열** (`text/template`) — 카탈로그를 외부 파일로 분리 가능
- C. **JSON/YAML 외부 카탈로그 + 변수 치환** — FR-7.2 외부화 가장 강함
- D. Other

> 비고: A를 선택해도 인터페이스(`AnnouncementCatalog`)로 추상화하면 추후 B/C로 전환 가능 (FR-7.2 보장).

[Answer]: A

---

### Q-FD-U2-8. 안내 카탈로그 톤 / 페르소나 (CR2 사용자 명시: "근엄한 톤, 한국어만")

- A. **근엄·차분·고전적 진행자** ("이제 밤이 깊어졌습니다…", "마을 사람들이여, 운명을 결정하시오") (권장)
- B. **친근·캐주얼 톤** ("자, 이제 밤이에요!")
- C. Other

[Answer]: A

---

### Q-FD-U2-9. SQLite 스키마 — 테이블 분할

- A. **3 테이블**: `active_snapshot` (단일 row, 진행 중 게임), `game_results` (완료된 게임 누적), `events` (선택, 디버깅용 이벤트 로그) — 권장
- B. **2 테이블**: `active_snapshot`, `game_results` (events 테이블 생략)
- C. **1 테이블 + JSON 컬럼** — 단순하지만 조회 비효율
- D. Other

[Answer]: A

---

### Q-FD-U2-10. SQLite 파일 위치 / 운영

- A. **`./data/mafia.db`** (워크스페이스 루트 기준 상대 경로). 디렉터리 자동 생성, 호스트 PC에서 단일 파일 (권장)
- B. **OS 표준 데이터 디렉터리** (예: `~/.local/share/mafia-game/mafia.db`) — 사용자별 격리
- C. **환경 변수 / CLI 플래그로 경로 지정** — 운영 유연
- D. Other

> 비고: A를 채택해도 옵션으로 환경변수/플래그 오버라이드 추가 가능 (코드 단계 결정).

[Answer]: A

---

### Q-FD-U2-11. JoinPlayer 정책 (FR-1.2)

- A. **LOBBY 단계에서만 입장 허용**, 진행 중 신규 입장 차단. 다만 같은 PlayerID(또는 토큰)로의 재연결은 단계 무관 허용 (권장)
- B. INTRO까지 입장 허용
- C. Other

[Answer]: A

---

### Q-FD-U2-12. 비공개 정보 영속화

스냅샷 BLOB에 Role/Keyword를 그대로 저장할지 여부:

- A. **그대로 저장** — 단일 바이너리·로컬 SQLite·사내 LAN 환경이라 디스크 보호 충분 (권장)
- B. **암호화 저장** — 추가 의존성·복잡도 증가, 본 PoC에는 과한 설계
- C. Other

[Answer]: A

---

### 자유 의견란

[Answer]: 

---

## 4. 분석 단계 (Step 5, 답변 후)

답변 수령 후 모호성 점검 → 필요 시 후속 질문 추가.

## 5. 산출물 미리보기 (Generation 시 작성)

- **`domain-entities.md`** — Session 구조, JoinPlayer 응답(token), AnnouncementCatalog 인터페이스, Snapshot record, GameResult record, SQLite 스키마(DDL 초안)
- **`business-logic-model.md`** — CreateSession/JoinPlayer/StartGame/SubmitAction/HostControl/Tick 의사 코드, 단계 전이 시 SaveSnapshot 트리거, 재연결 흐름, 안내 메시지 매핑 표
- **`business-rules.md`** — 권한·락·영속화·안내 매핑 규칙, 트랜잭션 경계, 에러 매핑
- **`frontend-components.md`** — N/A (U2 백엔드)
