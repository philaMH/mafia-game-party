# Unit of Work Plan — Mafia Game

**작성일**: 2026-04-26
**참조**: `requirements.md` v1.1, `execution-plan.md`, `application-design/*.md`
**프로젝트 유형**: Greenfield, 단일 Go 바이너리(React SPA `embed.FS` 동봉) — **Monolith 1개 배포 단위**

---

## 0. 본 단계의 목적

본 시스템은 **운영상 1개 배포 단위(단일 Go 바이너리)** 이지만, 개발 관점에서는 책임이 다른 모듈로 분리됩니다.
본 단계에서는 그 모듈을 **"Unit of Work" (개발 단위)** 로 정의하여, 후속 Construction 단계(Functional Design / NFR / Code Generation)가 단위별로 직렬 진행할 수 있도록 합니다.

> 용어: 본 프로젝트에서 "Unit of Work" = "개발 단위(논리적 모듈)" = Go의 internal 패키지 또는 React 영역. 각 Unit은 **독립 배포 단위가 아니며, 같은 단일 바이너리 내부의 모듈**입니다.

---

## 1. 작업 체크리스트 (Part 1 — Planning)

- [x] (1) 잠정 단위(U1~U6) 분할안 검토 및 확정 질문지 작성
- [x] (2) 단위 분할 결정 질문 작성 (아래 §3 참조)
- [x] (3) 본 plan 파일에 [Answer]: 태그 임베드
- [x] (4) 사용자 답변 수집 — Q-UG-1~6 모두 A
- [x] (5) 답변 모순/모호성 점검 — 일관성 확인, 후속 질문 불필요
- [x] (6) 사용자 승인 획득 — 2026-04-26
- [x] (7) Plan Part 2 진입 준비

### 분석 결과 (Step 7)
| 항목 | 결정 |
|---|---|
| 단위 입자도 | **5개 단위 채택** (U1~U5) |
| 프론트엔드 | `mafia-game/web/` 단일 Vite 프로젝트 + `web/dist/` `embed.FS` |
| Go 모듈 | **단일 go.mod**, 단위는 `internal/*` 패키지 |
| 단위 개발 순서 | **Bottom-up**: U1 → U2 → U3 → U4 → U5 |
| 공유 타입 정의처 | **U1 Game Core**가 도메인 타입의 단일 정의처 |
| 테스트 전략 | 단위 테스트(unit) + Build & Test 단계 통합 테스트 일괄 |

## 2. 작업 체크리스트 (Part 2 — Generation, 승인 후 실행)

- [x] (8) `aidlc-docs/inception/application-design/unit-of-work.md` 작성 — U1~U5 정의·책임·구성요소·코드 위치
- [x] (9) `aidlc-docs/inception/application-design/unit-of-work-dependency.md` 작성 — 의존 매트릭스 + Composition Root 통합 시퀀스
- [x] (10) `aidlc-docs/inception/application-design/unit-of-work-story-map.md` 작성 — FR/NFR ↔ 단위 매핑 (User Stories SKIP)
- [x] (11) Greenfield 코드 조직 전략 기록 (`unit-of-work.md` §0)
- [x] (12) 단위 경계·의존성 검증 (순환 의존 없음, 단방향 — `unit-of-work-dependency.md` §1)
- [x] (13) 모든 FR/NFR 단위 할당 검증 (`unit-of-work-story-map.md` §6)
- [x] (14) `aidlc-docs/aidlc-state.md` 갱신
- [x] (15) 사용자 승인 게이트 제시

---

## 3. 잠정 분할안 (검토 대상)

`execution-plan.md` §4의 U1~U6안을 Application Design의 9개 컴포넌트(C1~C9)에 매핑한 표:

| Unit | 단위명 | 포함 컴포넌트 | 코드 위치 (잠정) | 핵심 책임 |
|---|---|---|---|---|
| **U1** | Game Core (도메인) | C1 GameEngine, C2 RoleAssigner | `internal/game/*` | 상태 머신·역할/키워드 배분·규칙·종료 판정. 외부 I/O 0 |
| **U2** | Session & Persistence | C3 SessionManager, **C4 AnnouncementService**, C5 PersistenceStore | `internal/session/*`, `internal/announce/*`, `internal/persistence/*` | 단일 GM 락·세션 라이프사이클·이벤트→안내 메시지·SQLite 영속화 |
| **U3** | Realtime Transport | C6 WSHub | `internal/transport/ws/*` | WebSocket 연결/식별/라우팅·재연결 |
| **U4** | HTTP Bootstrap & Static | C7 HTTPServer | `cmd/mafia-game/main.go`, `internal/transport/http/*` | 부트스트랩·정적 자산(embed.FS)·라우팅·LAN IP 노출 |
| **U5** | Web Frontend (React SPA) | C8 PublicView, C9 PlayerView | `web/src/*` (Vite, TypeScript) | `/public`(TTS+자막) 및 `/play` 라우트, WebSocket 클라이언트, TTS 큐잉/폴백 |
| ~~U6~~ | (제거) | — | — | 잠정안의 U6은 U4와 통합 (단일 바이너리 부트스트랩 + 정적 자산은 한 단위가 자연스러움) |

> **잠정안의 변경점**:
> - **AnnouncementService(C4)** 는 SessionManager와 항상 함께 호출되며 별도 라이프사이클이 없으므로 **U2에 통합** (잠정안에는 미배정).
> - 잠정안의 U4(PublicView)+U5(PlayerView)를 **U5(Web Frontend) 1개**로 통합 (단일 React SPA, 같은 빌드/배포). 두 라우트는 코드 디렉터리(`web/src/views/Public`, `web/src/views/Player`)로만 분리.
> - 잠정안의 U6(HTTP Bootstrap)을 **U4**에 매핑 (이름만 정리).

---

## 4. 결정 질문 (사용자 답변 필요)

> **답변 방법**: 각 질문 아래 `[Answer]: <기호>`에 선택지를 적어 주세요. 자유 의견은 자유의견란에 적어 주세요.

### Q-UG-1. 단위 분할 입자도

§3 표의 **5개 단위(U1~U5)** 분할안을 채택할까요?

- A. 채택 (5개: Game Core / Session+Persistence+Announce / WS Transport / HTTP Boot+Static / Web Frontend)
- B. **AnnouncementService를 U2에서 분리**하여 별도 단위로 둔다 (총 6개)
- C. **PersistenceStore를 U2에서 분리**하여 별도 단위로 둔다 (총 6개) — 영속화 백엔드 교체 가능성을 강하게 분리하고 싶을 때
- D. 원안 (잠정 U1~U6) 그대로 유지: PublicView/PlayerView를 별개 단위로 (총 6개)
- E. **더 묶기**: U3(WS) + U4(HTTP) 통합 → 4개 단위
- F. Other (자유의견란에 기술)

[Answer]: A 

---

### Q-UG-2. 프론트엔드 단위 구성

React SPA를 어떻게 구성/빌드하나요? (이는 모노레포 구조와 빌드 산출물 위치에 영향)

- A. `mafia-game/web/` 단일 Vite 프로젝트, 빌드 산출물 `web/dist/`를 Go가 `//go:embed web/dist` 로 동봉 (Application Design `components.md`의 잠정안)
- B. 동일하지만 React 소스를 `frontend/`로 두는 등 디렉터리 이름 변경
- C. Other (자유의견란)

[Answer]: A

---

### Q-UG-3. Go 모듈 구조

- A. **단일 Go 모듈** (`go.mod` 1개, `module github.com/<owner>/mafia-game`) — 권장. 단위는 `internal/*` 패키지로만 분리
- B. **Go workspace + 다중 모듈** (각 unit이 독립 go.mod) — 과도한 분할 위험, 작은 도구 규모상 비권장
- C. Other (자유의견란)

[Answer]: A

---

### Q-UG-4. 단위 개발 순서 (Construction 페이즈 진행 순서)

Construction 페이즈는 단위별 루프(Functional Design → NFR Req → NFR Design → Code Gen)를 단위마다 수행합니다. 어느 순서로 진행할까요?

- A. **의존성 역순(=Bottom-up)**: U1 Game Core → U2 Session+Persistence → U3 WS Transport → U4 HTTP Boot → U5 Web Frontend (의존성이 적은 단위부터 — 권장)
- B. **End-to-end 우선**: U4 HTTP+Static + 최소 U5 → 점차 깊이로 (외부 인터페이스 먼저 — 데모 가시성 좋음)
- C. **수직 슬라이스**: 게임 시작/종료/투표 등 시나리오별로 U1~U5를 횡단 (이번 워크플로우는 단위 단위 루프이므로 비호환)
- D. Other (자유의견란)

[Answer]: A

---

### Q-UG-5. 단위 간 인터페이스 정의 위치

단위 간 공유 타입(예: `PlayerID`, `Event`, `Action`)을 어디에 둘까요?

- A. **U1 Game Core가 도메인 타입의 단일 정의처**, 다른 단위는 import (Application Design `component-methods.md`의 공용 타입과 일치 — 권장)
- B. 별도 `internal/types` 패키지에 모음
- C. 각 단위가 자체 정의 + 변환 어댑터 (가장 분리되지만 비용 높음)
- D. Other

[Answer]: A

---

### Q-UG-6. 단위별 테스트 책임

각 단위의 테스트 가이드라인:

- A. **단위마다 단위 테스트(Go: `_test.go` / React: Vitest) + Build & Test 단계에서 통합 테스트 일괄** — 권장
- B. 통합 테스트도 단위별로 분산
- C. Other

[Answer]: A

---

### 자유 의견란

- 추가 분할/통합 제안, 단위 이름 변경, 우선순위 등 자유롭게 적어주세요.

[Answer]: 

---

## 5. 분석 단계 (Step 7~8, 답변 수신 후 자동 수행)

답변 수신 후 다음을 점검합니다:
- 모호한 응답("mix of", "depends" 등) 및 모순 답변 검출
- 답변에 따라 §3 표를 갱신하고 본 plan 체크리스트 (1)~(2)을 [x] 처리
- 필요 시 후속 질문을 동일 형식으로 본 문서 하단에 추가

---

## 6. 승인 게이트 (Step 9)

분석 완료 후 다음 메시지를 제시합니다:
> **Unit of work plan complete. Review the plan in `aidlc-docs/inception/plans/unit-of-work-plan.md`. Ready to proceed to generation?**

---

## 7. Generation 산출물 미리보기 (Part 2 시 작성)

- `aidlc-docs/inception/application-design/unit-of-work.md`
  - 각 단위: ID/이름, 목적, 포함 컴포넌트, 코드 위치, 외부 의존, 인터페이스 요약, 빌드 산출물
- `aidlc-docs/inception/application-design/unit-of-work-dependency.md`
  - 의존 매트릭스 (Mermaid + 텍스트 대안)
  - 통합 시퀀스 (예: 부팅 시 와이어링 순서)
- `aidlc-docs/inception/application-design/unit-of-work-story-map.md`
  - FR/NFR ↔ 단위 매핑 (User Stories SKIP이므로 requirements.md v1.1 항목으로 매핑)
