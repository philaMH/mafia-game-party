# NFR Requirements Plan — U5 Web Frontend

**작성일**: 2026-04-26
**대상 단위**: U5 / Web Frontend (`web/src/*`)
**참조**:
- `requirements.md` v1.1 NFR-2/3/4/6/7
- `aidlc-docs/construction/u5-web-frontend/functional-design/*.md`

> 본 plan은 U5 NFR Requirements의 단일 진실 소스입니다.

---

## 0. NFR 영역 우선순위

U5는 프레젠테이션 단위. 비즈니스 로직 0.

| 영역 | 적용 여부 | 비고 |
|---|:---:|---|
| Performance | **상위** | LAN 즉시 반응, TTS 지연 최소화 |
| Usability | **상위** | NFR-3 한국어, 가독성, 모바일/태블릿 반응형 |
| Reliability | 중간 | WS 재연결, TTS 폴백 |
| Maintainability | **상위** | TS strict, 컴포넌트 단위 테스트 |
| Security | 중간 | 토큰 비공개, 마스킹 (백엔드 보장에 의존) |
| Accessibility | 중간 | 자막 폴백, 큰 폰트 |
| Scalability / Availability | **N/A** | 단일 호스트 PC LAN |

---

## 1. 단계 체크리스트

- [x] (1) 영역 우선순위 평가
- [x] (2) 결정 질문 작성 (Q-NFR-U5-1~12)
- [x] (3) plan 문서 작성 (본 파일)
- [x] (4) 사용자 답변 수집 — "승인" (권장 답안)
- [x] (5) 답변 일관성 검증
- [x] (6) `nfr-requirements.md` 작성
- [x] (7) `tech-stack-decisions.md` 작성
- [x] (8) audit + aidlc-state 갱신, 사용자 승인 게이트

---

## 2. 결정 질문 (Q-NFR-U5-1 ~ Q-NFR-U5-12)

### Q-NFR-U5-1. 외부 의존성

U5의 npm 직접 의존은?

- **A.** **`react`, `react-dom`, `react-router-dom`, `vite`, `typescript`, `@types/react`, `@types/react-dom`** + 테스트(`vitest`, `@testing-library/react`, `@testing-library/jest-dom`, `jsdom`). 추가 lib 0.
- **B.** + Tailwind / shadcn UI.
- **C.** + Redux / Zustand.

[Answer]: A

### Q-NFR-U5-2. WS 메시지 → UI 반영 지연

LAN 환경에서 wire 메시지 수신 → 화면 갱신까지 지연 목표는?

- **A.** **p99 < 100 ms** (React 리렌더 + setState).
- **B.** p99 < 50 ms (엄격).
- **C.** p99 < 200 ms (느슨).

[Answer]: A

### Q-NFR-U5-3. TTS 발화 지연

`announce` 수신 → SpeechSynthesis 발화 시작 지연은?

- **A.** **p99 < 200 ms** (Web Speech API 큐잉 포함).
- **B.** p99 < 500 ms.
- **C.** 측정 안 함.

[Answer]: A

### Q-NFR-U5-4. TS strict 모드

TypeScript 설정?

- **A.** **`"strict": true` + `"noUncheckedIndexedAccess": true`** — 타입 안전 최대화.
- **B.** strict만.
- **C.** strict 비활성.

[Answer]: A

### Q-NFR-U5-5. 단위 테스트 커버리지 목표

핵심 모듈(reducer, useTTSQueue, NicknameForm validation)의 라인 커버리지 목표는?

- **A.** **≥ 70%** — UI 컴포넌트는 일부만 단위 테스트, 통합 테스트는 PoC 범위 외.
- **B.** ≥ 85%.
- **C.** ≥ 50%.

[Answer]: A

### Q-NFR-U5-6. 빌드 산출물 크기

`web/dist` 총 크기 한도는?

- **A.** **gzip < 500 KB** (React + Router + 자체 코드).
- **B.** gzip < 1 MB.
- **C.** 한도 없음.

[Answer]: A

### Q-NFR-U5-7. 모바일 / 태블릿 반응형

PlayerView의 최소 지원 화면 폭은?

- **A.** **320px (iPhone SE)** — 모바일 우선.
- **B.** 768px (태블릿만).
- **C.** 1024px (데스크톱만).

[Answer]: A

### Q-NFR-U5-8. PublicView 가독성

공용 화면의 자막/플레이어 카드 폰트 크기는?

- **A.** **자막 ≥ 32px / 단계 헤더 ≥ 48px / 플레이어 닉네임 ≥ 24px** (FD §3 설계).
- **B.** 모두 16px (브라우저 기본).
- **C.** 사용자 슬라이더로 조절.

[Answer]: A

### Q-NFR-U5-9. ESLint / 정적 분석

코드 품질 도구는?

- **A.** **`eslint` + `@typescript-eslint/parser` + `eslint-plugin-react-hooks`** (Vite 기본 lint 설정).
- **B.** + Prettier 자동 포맷.
- **C.** lint 미적용.

[Answer]: A

### Q-NFR-U5-10. 접근성 (a11y)

a11y 수준 목표는?

- **A.** **자막 폴백 + 키보드 네비게이션 가능** (NFR-U5-A1) — WCAG AA 명시 추구하지는 않음 (PoC).
- **B.** WCAG AA 준수.
- **C.** 미고려.

[Answer]: A

### Q-NFR-U5-11. 토큰 노출 검증

`localStorage.mafia.token`이 화면/로그/URL에 노출되지 않음을 어떻게 검증?

- **A.** **단위 테스트**: NicknameForm, YourInfoCard 등의 렌더 결과를 querySelector로 grep — token 미포함 확인.
- **B.** + E2E 테스트.
- **C.** 코드 리뷰만.

[Answer]: A

### Q-NFR-U5-12. WS 재연결 한도

지수 백오프 상한 시간은?

- **A.** **상한 16초** (FD §3.1) — 그 이상은 그대로 16초 간격으로 무한 재시도.
- **B.** 상한 60초 후 포기.
- **C.** 무한 재시도, 백오프 없음.

[Answer]: A

---

## 3. 산출물 예상

| 파일 | 책임 |
|---|---|
| `nfr-requirements.md` | 6개 영역 NFR + 트레이드오프 + FR/NFR 추적성 + 검증 게이트 + 비-요구사항 |
| `tech-stack-decisions.md` | npm 직접 의존 7종 + Vite + TS strict + 패키지 레이아웃 |

---

## 4. 사용자 승인 게이트

본 plan과 답변을 검토해 주세요. 모든 답변에 동의하시면 **"완료"** 또는 **"승인"** 으로 응답해 주세요.
