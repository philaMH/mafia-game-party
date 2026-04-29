# Iteration 7 — Requirements (호스트 첫 페이지 / Host Main Menu)

- **버전**: v1.0 (사용자 승인 대기)
- **작성일**: 2026-04-29
- **유형**: Brownfield Patch (Iteration 7)
- **추적 입력**: `iteration7-intake-questions.md` 8문항 답변 (Q1=A, Q2=C, Q3=B, Q4=A, Q5=B, Q6=A, Q7=A, Q8=C)

## 1. 개요

기존에는 호스트가 `/public` 라우트에 접속하면 `host claim` 후 곧바로 옵션 입력 + "♠ 방 개설" 버튼이 한 화면에 노출되는 단일 진입 구조였다(`web/src/views/PublicView/PublicView.tsx:134-189`). 본 이터레이션은 그 진입점을 **메인 메뉴(2 버튼)**와 **별도 설정 라우트**로 분리한다. 옵션은 클라이언트 `localStorage`에 영속화하고 서버에도 사전 저장(신규 wire `host:save-options`)함으로써, 호스트가 "게임 시작"을 누르면 별도 옵션 입력 없이 즉시 LOBBY가 개설된다.

## 2. 기능 요구사항 (Functional Requirements)

### FR-1. 호스트 메인 페이지 (Q1=A, Q3=B, Q6=A)
- 호스트가 `/public` 라우트에서 `host:claim` 성공 후 `roomOpened === false` 상태일 때 노출되는 화면.
- 화면 구성: MAFIA 타이틀(노이르 스타일 유지) + 다음 2개 버튼.
  - **① 게임 시작** — 클릭 시 즉시 `host:open-room` 송신. payload `options`는 후술 FR-3 의 "현재 유효 옵션"을 사용. 화면 전환은 LOBBY 진입(기존 흐름과 동일).
  - **② 설정** — 클릭 시 `/public/settings` 라우트로 이동.
- 게임 종료(EndScreen 닫힘) 후 메인 메뉴로 복귀했을 때도 동일 메뉴(추가 결과 배너 없음). 호스트는 "게임 시작"으로 재차 LOBBY를 열 수 있다.

### FR-2. 설정 라우트 `/public/settings` (Q2=C, Q3=B, Q4=A, Q7=A)
- React Router 신규 경로 `/public/settings` 추가.
- 화면 구성:
  - 노이르 스타일 패널에 다음 9개 필드 모두 노출.
    1. 최대 참여 인원 `maxPlayers` (6~12, number)
    2. 마피아 수 `mafiaCount` (1 ~ `maxPlayers - 3`, number)
    3. 자기소개 시간 `introSecondsPerPlayer` (초, number)
    4. 토론 시간 `discussionSeconds` (초, number)
    5. 마피아 시간 `nightMafiaSeconds` (초, number)
    6. 경찰 시간 `nightPoliceSeconds` (초, number)
    7. 의사 시간 `nightDoctorSeconds` (초, number)
    8. 의사 자가치료 허용 `doctorSelfHealAllowed` (체크박스)
    9. 음성 안내 `announcementVoiceOn` (체크박스)
  - 하단에 단일 버튼 **"저장 후 메인으로"** 1개. 클릭 시 다음을 한 트랜잭션으로 수행:
    - (a) 클라이언트 `localStorage`에 직렬화 저장 (FR-4)
    - (b) 서버에 `host:save-options` wire 송신 (FR-5)
    - (c) `/public` 라우트(메인 메뉴)로 라우팅
- 권장값 가이드는 기존과 동일하게 `mafiaCount`가 `defaultOptions(maxPlayers).mafiaCount` 와 1 이상 차이날 때 인라인 경고("※ 권장하지 않는 설정입니다") 표시. 저장은 막지 않음(Q7=A).

### FR-3. 현재 유효 옵션 (Effective Options)
- 메인 메뉴 "게임 시작"이 `host:open-room`에 실어 보내는 옵션은 다음 우선순위로 결정:
  1. 호스트가 본 세션에서 설정 화면을 통해 마지막으로 저장한 값
  2. 1이 없으면 `localStorage`에서 복원한 값 (FR-4)
  3. 둘 다 없으면 `defaultOptions(8)` (현 코드 기본값과 동일)
- "설정 화면 진입 시 보여주는 초기값"도 동일한 우선순위로 결정.

### FR-4. 클라이언트 영속 (Q5=B)
- 키: `mafia.options.v1` (스키마 변경 시 v2로 bump 가능하도록 버전 suffix 부여).
- 값: `JSON.stringify(Options)` 형식.
- 읽기 시점: `GameProvider` 마운트 또는 설정 화면 진입 시 1회.
- 쓰기 시점: 설정 화면 "저장 후 메인으로" 버튼 클릭 시.
- 파싱 실패/스키마 불일치/필드 누락 시: 안전하게 무시하고 `defaultOptions(8)` 적용 + `localStorage` 키 삭제.
- 다른 브라우저/디바이스 간 동기화는 범위 외.

### FR-5. 서버 영속 — 신규 wire 메시지 (Q8=C)
- 신규 outgoing 메시지: `{ type: "host:save-options", options: Options }`
  - 송신 트리거: 설정 화면 "저장 후 메인으로" 클릭, 호스트 인증 상태에서만.
  - 서버 처리: 호스트 토큰 보유자에 한해 옵션을 서버 측 메모리(또는 기존 `Session` 컨테이너 적합 위치)에 보관. 영속 정책은 functional design에서 결정(인메모리 우선, persistence 모듈 활용 여부는 design 단계 판단).
- 신규 incoming 메시지: 별도로 정의하지 않는다(클라이언트는 송신 후 응답을 기다리지 않음). 단, 검증 실패 시 기존 에러 채널(`error` 토스트)로 회신해야 한다.
- `host:open-room`은 기존대로 `options` 페이로드를 그대로 가진다(하위 호환). 서버는 `host:open-room`이 도착하면 그 페이로드를 단일 진실 소스로 사용한다. `host:save-options`는 호스트 재접속(재로그인) 시 옵션을 복원하기 위한 사전 저장 용도이며, 옵션 동기화의 보조 채널이다.

### FR-6. 호스트 재접속 시 옵션 복원
- 호스트가 토큰을 보유한 상태로 재접속하면, 서버가 `host:save-options`로 받은 마지막 옵션을 호스트 클라이언트에 다시 노출할 수 있어야 한다.
- 본 이터레이션의 최소 동작:
  - 클라이언트는 `localStorage` 우선 사용(FR-3), 서버 측 복원은 design 단계에서 protocol 선택(예: 기존 `host:claim` 응답 또는 `state` 첫 push에 옵션 인클루드).
- 본 이터레이션 범위에서는 클라이언트 `localStorage`만으로도 UX가 성립해야 한다. 서버 영속의 사용은 옵션이며, design 단계에서 노출 형식을 확정한다.

## 3. 비기능 요구사항 (NFR)

- **NFR-1 성능**: 메인 메뉴/설정 화면 모두 SSR 없는 SPA 라우트 전환만 수행. 상태 변경은 기존 reducer 스캐폴드 안에 머무름.
- **NFR-2 호환성**: 비-호스트(참가자 또는 PUBLIC 관전자)는 `/public/settings` 직접 진입 시 메인으로 리다이렉트(또는 안전한 placeholder). `/public` 비-호스트 진입 시 동작은 기존과 동일(LOBBY 자막 등).
- **NFR-3 안정성**: `localStorage` 비활성/풀 상태/JSON parse 실패에서도 앱은 크래시 없이 기본값으로 동작.
- **NFR-4 i18n / 노이르 톤**: 새 화면은 기존 `noir.css`(Iteration 6) 토큰을 재사용. 별도 폰트/팔레트 추가 없음.
- **NFR-5 테스트**: 다음을 신규 포함.
  - 메인 메뉴 렌더 + 두 버튼 기능 (vitest)
  - 설정 화면 9 필드 입력 + 저장 액션 (vitest)
  - `localStorage` 라운드트립 (vitest, jsdom 모킹)
  - 서버측 `host:save-options` dispatch 핸들러 (Go 단위 테스트)
- **NFR-6 보안**: `host:save-options`는 호스트 토큰 검증 후에만 처리. 옵션 값은 기존 `Options` validation 범위(예: 음수/과대값 거부)를 통과해야 함.

## 4. 영향 단위 (Impact Map)

| 단위 | 영향 | 비고 |
|---|---|---|
| U1 Game Core | (없음 또는 미미) | `Options` 검증 강화는 자연스러우나 본 이터레이션 강제 사항 아님. |
| U2 Session/Persistence/Announce | 보통 | 호스트별 옵션 보관 위치(메모리/persistence) 결정 + restore 시 노출. |
| U3 Realtime Transport | 보통 | 신규 incoming wire `host:save-options` + 디스패치 추가. |
| U4 HTTP Bootstrap | 없음 | 정적 자산 자동 갱신만. |
| U5 Web Frontend | 큼 | 신규 라우트/뷰 2개, `localStorage` 모듈, GameContext의 옵션 노출 보강. |

## 5. 수용 기준 (Acceptance Criteria)

- AC-1: 호스트가 `/public` 진입 시 메인 메뉴(MAFIA 타이틀 + "게임 시작" + "설정")가 보인다.
- AC-2: "설정" 클릭 시 `/public/settings` 라우트로 이동하고 9개 필드가 현재 유효 옵션 값으로 미리 채워져 있다.
- AC-3: 설정 화면에서 값 변경 후 "저장 후 메인으로" 클릭 시 `/public` 으로 복귀하며, 다시 "설정" 진입 시 변경한 값이 유지된다.
- AC-4: 새로고침 후 다시 호스트로 진입해도 마지막 저장한 옵션이 메인 메뉴/설정에 그대로 보인다.
- AC-5: "게임 시작" 클릭 시 `host:open-room`이 현재 유효 옵션과 함께 송신되고, 서버는 LOBBY를 연다.
- AC-6: 설정 화면에서 권장 범위(±1)를 벗어난 `mafiaCount` 입력 시 인라인 경고가 표시되지만 저장은 가능하다.
- AC-7: 서버가 `host:save-options`를 호스트 토큰과 함께 수신하면 옵션을 보관하고, 그렇지 않은 클라이언트가 보내면 거부 + error 회신.
- AC-8: 비-호스트가 `/public/settings` 직접 진입 시 메인으로 리다이렉트된다.

## 6. 명시적 비기능 (Out of Scope)

- 옵션 프리셋(쉬움/보통/어려움) 추가는 범위 외.
- 다중 호스트 / 옵션 동시 편집 충돌 처리는 범위 외(호스트는 1명만이라는 기존 invariant 유지).
- 다국어 라벨 추가/i18n 프레임워크는 범위 외(기존과 동일한 한국어 라벨).
- 옵션 변경 이력(audit log) 노출은 범위 외.

## 7. Extension Configuration 영향

| Extension | Enabled | 본 이터레이션 영향 |
|---|---|---|
| Security Baseline | No | 변경 없음. `host:save-options` 호스트 토큰 검증은 기존 인증 체계 재사용. |

## 8. 모호성/리스크 노트

- (R-1) Q5(클라이언트 localStorage) + Q8(서버 `host:save-options`) 동시 채택으로 인해 "옵션의 진실 소스" 우선순위는 본 문서 §FR-3 에서 클라이언트 우선으로 명시함. design 단계에서 서버 복원 protocol 정의 필요.
- (R-2) 서버측 옵션 보관 위치(인메모리 vs persistence 모듈) 선택은 design 단계에서 결정 — 현재는 호스트 1명/단일 게임 invariant 이므로 인메모리로 충분히 대응 가능.

## 9. 사용자 승인 (Approval Gate)

본 요구사항 문서를 검토하시고 다음 중 하나로 응답해 주십시오.

- **승인** — Workflow Planning 단계로 진행
- **수정** — 변경/보완할 항목을 알려주시면 v1.1로 갱신
