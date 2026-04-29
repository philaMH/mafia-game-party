# Application Design Plan — Mafia Game

**작성일**: 2026-04-25
**참조**: `requirements.md` v1.1, `execution-plan.md`

본 문서는 Application Design 단계의 작업 계획과, 컴포넌트 설계에 영향을 주는 결정 질문을 포함합니다.

---

## 1. 작업 체크리스트

- [x] 요구사항(`requirements.md` v1.1) 및 실행 계획(`execution-plan.md`) 분석
- [x] **결정 질문 사용자 답변 수집** (본 문서 §3) — Q-AD-1=A(SQLite), Q-AD-2=A(gorilla), Q-AD-3=C+React, Q-AD-4=B, Q-AD-5=C, Q-AD-6=B, Q-AD-7=Other(오프라인 협의 후 마피아 한 명이 입력), Q-AD-8=A
- [x] `application-design/components.md` 작성 — 컴포넌트 정의 및 책임
- [x] `application-design/component-methods.md` 작성 — 컴포넌트별 메서드 시그니처
- [x] `application-design/services.md` 작성 — 서비스 정의 및 오케스트레이션
- [x] `application-design/component-dependency.md` 작성 — 컴포넌트 의존성/통신 패턴
- [x] `application-design/application-design.md` 작성 — 통합 설계 문서
- [x] 설계 일관성·완결성 검증 — application-design.md §8 체크리스트 충족
- [ ] 사용자 승인 (대기 중)

---

## 2. 컴포넌트 설계 방향 (잠정)

execution-plan.md §4에 잠정 Units 후보(U1~U6)를 제시했습니다. Application Design은 이를 기반으로 다음 컴포넌트를 정의할 예정입니다 (이름은 잠정):

| 분류 | 컴포넌트 (잠정) | 책임 (요약) |
|---|---|---|
| 도메인 | **GameEngine** | 상태 머신, 역할/투표/종료 판정, 키워드 부여 |
| 도메인 | **RoleAssigner** | 인원수 기반 역할 배분, 키워드 풀에서 무작위 추출 |
| 애플리케이션 | **SessionManager** | 단일 게임 세션 라이프사이클, 단일 GM 락 |
| 애플리케이션 | **AnnouncementService** | 단계 전환·결과·진행자 멘트 안내 메시지 생성/디스패치 (FR-8 카탈로그 매핑) |
| 인프라 | **PersistenceStore** | 게임 상태 영속화/복원, 결과 저장 |
| 인프라 | **WSHub** | WebSocket 연결 관리, 클라이언트별 라우팅, 재연결 |
| 인프라 | **HTTPServer** | HTTP 라우팅, 정적 자산 서빙, IP 노출 |
| 프레젠테이션 | **PublicView** (브라우저) | 공용 화면 렌더링, Web Speech TTS 큐잉/인터럽션, 자막, 토글 |
| 프레젠테이션 | **PlayerView** (브라우저) | 자기 역할/키워드/밤 행동/투표 입력 (반응형) |

세부 메서드/시그니처/의존성은 사용자 답변 후 산출물에서 구체화합니다.

---

## 3. 결정이 필요한 질문 (사용자 답변 부탁드립니다)

> 각 질문 아래의 `[Answer]:` 옆에 알파벳을 적어주시고, 모두 답변하신 후 "완료"라고 알려주세요.

### Q-AD-1: 영속화 백엔드
요구사항 NFR-1·NFR-7에 따라 **호스트 PC 재시작 시에도 진행 중 게임 복원** 이 가능해야 합니다. 어떤 영속화 방식을 사용할까요?

A) **SQLite** (`mattn/go-sqlite3` 또는 순수 Go `modernc.org/sqlite`) — 표준 SQL, 결과 통계 확장 시 유리. 단 cgo 또는 추가 의존성 필요
B) **BoltDB / bbolt** (`go.etcd.io/bbolt`) — 순수 Go 임베디드 KV, 단일 파일, 의존성 가벼움
C) **JSON 스냅샷 파일** (`encoding/json` + 주기적 fsync) — 가장 단순, 단 동시성·무결성 직접 관리 필요
D) AI 추천 — 근거와 함께 자동 결정
E) Other (please describe after [Answer]: tag below)

[Answer]: A) 

---

### Q-AD-2: WebSocket 라이브러리
NFR-2(상태 동기화 < 1s)와 NFR-1(재연결) 요구를 충족하기 위한 WebSocket 라이브러리는?

A) **`gorilla/websocket`** — 가장 널리 쓰임, 안정적, 풍부한 예제 (커뮤니티 유지보수)
B) **`coder/websocket`** (구 `nhooyr.io/websocket`) — `context` 통합·테스트 친화, 모던한 API
C) **표준 `net/http` + 직접 업그레이드 구현** — 외부 의존 최소화, 단 직접 구현 부담↑
D) AI 추천
E) Other (please describe after [Answer]: tag below)

[Answer]: A)

---

### Q-AD-3: 프론트엔드 형태
공용 화면(`PublicView`) + 개인 화면(`PlayerView`)의 UI 구현은?

A) **바닐라 JS + HTML 템플릿** — 빌드 도구 없이 단일 바이너리에 정적 자산 동봉(`embed.FS`). 가장 단순하고 빠른 시작
B) **경량 프레임워크 (Alpine.js / htmx 등)** — 작성성 향상, 빌드 단계 거의 없음
C) **SPA 프레임워크 (Vue / Svelte / React 미니 셋업)** — 컴포넌트화 강력, 빌드 도구(Vite 등) 필요
D) AI 추천
E) Other (please describe after [Answer]: tag below)

[Answer]: C

---

### Q-AD-4: 자기소개 단계 진행 방식
FR-4.1 자기소개 단계의 종료 시점은 어떻게 결정하나요?

A) **호스트가 "다음" 클릭으로 한 명씩 진행** — 호스트가 공용 화면에서 "다음 발화자" 버튼을 눌러 순서대로 진행. 키워드 발화·자기소개가 자유롭게 끝났을 때 인간이 판단
B) **시간 기반 자동 진행** — 사람당 N초(예: 30초) 타이머, 끝나면 자동으로 다음 사람. 시간은 게임 옵션으로 노출
C) **두 모드 모두 지원** — 호스트가 시작 시 선택
D) Other (please describe after [Answer]: tag below)

[Answer]: B)

---

### Q-AD-5: 토론 단계 종료 방식
FR-4.3 낮의 토론 단계는 어떻게 종료되어 투표로 넘어가나요?

A) **고정 타이머** (예: 90초) 종료 시 자동 투표 진입. 잔여 30초 음성 안내(FR-8.4)
B) **호스트 수동 종료** — 호스트가 "투표 시작" 버튼을 누르면 종료
C) **타이머 + 호스트 조기 종료 가능** — 기본 타이머가 흐르되 호스트가 언제든 단축 가능
D) Other (please describe after [Answer]: tag below)

[Answer]: C

---

### Q-AD-6: 호스트 권한
호스트(서버 실행자)는 누구이며 권한 범위는?

A) **호스트 = 게임에 참여하지 않는 별도 공용 화면 운영자** — 호스트는 플레이어가 아님 (단, 요구사항 §1.2 "모두가 플레이어"와 충돌 가능)
B) **호스트도 플레이어** (공용 화면을 운영하면서 본인도 역할 수령) — 단, 비공개 정보를 자기 폰에서 별도 확인. 호스트만의 추가 권한(시작/일시정지/강제종료 등)은 공용 화면에서 가능
C) **공용 화면 페이지에 들어간 클라이언트가 곧 호스트** (인증 없이 첫 접속자 또는 별도 라우트 `/public`)이며 플레이는 안 함. 다른 사람들이 모두 플레이어
D) Other (please describe after [Answer]: tag below)

[Answer]: B

---

### Q-AD-7: 마피아가 여러 명일 때의 야간 살해 합의
FR-4.2에서 마피아 다수가 행동할 때 살해 대상은 어떻게 결정?

A) **다수결** — 각 마피아가 표를 던지고, 시간 종료 시 최다 득표자 살해. 동률은 무작위 1명
B) **단일 합의(만장일치)** — 모든 마피아가 같은 사람을 지목해야만 살해 발생. 합의 못 하면 그날 살해 없음
C) **최후 입력자 우선** — 마지막에 입력한 마피아의 선택으로 결정 (가장 단순)
D) **다수결 + 동률 시 무작위, 시간 초과 시 무살해**
E) Other (please describe after [Answer]: tag below)

[Answer]: 오프라인에서 협의 후 마피아 대표자 (임의 선정) 가 자기 폰에서 선택함

---

### Q-AD-8: 의사 자가 보호 허용 여부
FR-2.1 의사가 자기 자신을 보호 대상으로 선택할 수 있나요?

A) **허용** (1회/매 밤 자기 자신 가능)
B) **불허** (자신 외에만 선택 가능)
C) **제한적 허용** — 게임 전체에서 자기 자신을 보호한 횟수에 제한 (예: 1회만)
D) Other (please describe after [Answer]: tag below)

[Answer]: A)

---

## 4. (선택) 추가 메모
컴포넌트 명명 선호, 도메인 패키지 구조 선호, 테스트 전략 선호 등 자유롭게.

[Free-form notes]: 
