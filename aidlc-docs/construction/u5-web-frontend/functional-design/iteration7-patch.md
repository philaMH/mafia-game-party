# U5 Functional Design — Iteration 7 Patch (Host Main Menu + Settings Route)

- **버전**: v1.0
- **작성일**: 2026-04-29
- **유형**: Brownfield Patch (UI 분리 + 신규 라우트 + localStorage 영속화)
- **추적 입력**: `inception/requirements/iteration7-requirements.md` v1.0 §FR-1~FR-6, `construction/u3-realtime-transport/functional-design/iteration7-patch.md` v1.0
- **상위 단계**: Iteration 1~6 산출물 보존, 본 패치는 신규 뷰/라우트/모듈 + 기존 PublicView 진입 분기 변경

## 1. 변경 개요

호스트가 `/public` 진입 시 보던 단일 화면(옵션 + "♠ 방 개설")을 **메인 메뉴(2 버튼)**와 **별도 설정 라우트(`/public/settings`)**로 분리한다. 옵션은 `localStorage`에 영속화하며, 저장 시 서버에도 신규 wire `host:save-options`로 사전 저장한다. "게임 시작"은 즉시 `host:open-room`을 송신하고, 페이로드 옵션은 다음 우선순위로 결정한다 — (1) 본 세션에서 마지막 저장값, (2) localStorage 복원값, (3) `defaultOptions(8)`.

## 2. 인터페이스 변경

### 2.1 신규 라우트 (App.tsx)

```tsx
<Route path="/public" element={<PublicView />} />
<Route path="/public/settings" element={<HostSettingsView />} />
```

`/public/settings` 진입 시 호스트가 아니거나 host claim 직후가 아니면 `<Navigate to="/public" replace />`로 리다이렉트(NFR-2 호환).

### 2.2 신규 모듈 — `web/src/lib/optionsStorage.ts`

```ts
import type { Options } from "../types/wire";

const KEY = "mafia.options.v1";

export function loadSavedOptions(): Options | null { /* ... */ }
export function saveOptions(opts: Options): void { /* ... */ }
export function clearSavedOptions(): void { /* ... */ }
```

- 파싱 실패/스키마 누락 필드 발견 시 `null` 반환 + 키 삭제(FR-4 안전 페일백).
- `safeLocalStorage()` 패턴(useToken 참고) 재사용.

### 2.3 GameContext 확장 (additive)

`GameContextValue`에 옵션 헬퍼 노출:

```ts
hostOptions: Options;        // 현재 유효 옵션 (FR-3 우선순위 적용 결과)
saveHostOptions(opts: Options): void;  // localStorage + send({type:"host:save-options", options})
```

내부 구현:
- `useState<Options>(() => loadSavedOptions() ?? defaultOptions(8))` — 한 번만 초기화.
- `saveHostOptions`는 localStorage 기록 + 메모리 업데이트 + ws 송신을 한 함수에서 처리.

추가 reducer 변경 없음(서버는 outgoing ack를 내지 않으므로 reducer 액션 추가 불필요).

### 2.4 신규 컴포넌트 (web/src/views/PublicView/)

| 컴포넌트 | 역할 |
|---|---|
| `HostHomeView` | 메인 메뉴 화면(타이틀 + "♠ 게임 시작" + "⚙ 설정" 두 버튼) |
| `HostSettingsView` | 9 필드 입력 폼 + "저장 후 메인으로" 단일 버튼 |

`PublicView.tsx`는 호스트 토큰 + 미개설 분기에서 `<HostHomeView />`를 렌더하도록 단순화한다(현재의 인라인 폼 제거).

### 2.5 신규 OutgoingMsg variant (`types/wire.ts`)

```ts
| { type: "host:save-options"; options: Options }
```

## 3. 동작 (Behavior)

### 3.1 메인 메뉴 (HostHomeView)
- 진입 시 `ctx.hostOptions`가 이미 결정되어 있음.
- "♠ 게임 시작" 클릭 → `ctx.send({ type: "host:open-room", options: ctx.hostOptions })` (FR-1).
- "⚙ 설정" 클릭 → `useNavigate()`로 `/public/settings`로 이동.
- 노이르 톤 유지(`mafia-title.stone`, `btn-noir.primary`, `gold-frame`).

### 3.2 설정 화면 (HostSettingsView)
- 진입 시 `ctx.hostOptions`로 폼 초기값 채움. `useState<Options>(ctx.hostOptions)`.
- 9 필드:
  - 숫자 7개: `maxPlayers / mafiaCount / introSecondsPerPlayer / discussionSeconds / nightMafiaSeconds / nightPoliceSeconds / nightDoctorSeconds`
  - 체크박스 2개: `doctorSelfHealAllowed / announcementVoiceOn`
- 권장값 가이드: `mafiaCount`가 `defaultOptions(maxPlayers).mafiaCount` 와 1 이상 차이날 때 `※ 권장하지 않는 설정입니다` 인라인 경고(현재 PublicView 텍스트 그대로 재사용).
- 단일 버튼 **"저장 후 메인으로"** 클릭 시:
  1. `ctx.saveHostOptions(form)` 호출 — localStorage 기록 + ws 송신.
  2. `useNavigate()`로 `/public`으로 이동.
- 비-호스트 진입 차단: `useEffect`에서 `!ctx.hostToken` 또는 `ctx.roomOpened` 시 `<Navigate replace to="/public" />` 반환.

### 3.3 GameContext의 `saveHostOptions`
1. `setHostOptions(opts)`
2. `saveOptions(opts)` — localStorage 기록.
3. `send({ type: "host:save-options", options: opts })` — ws 송신. 서버 응답은 처리하지 않음(에러 시 기존 `error` 토스트 채널이 자동 노출).

### 3.4 PublicView 진입 분기 변경
기존 라인 134~189 인라인 폼을 제거하고 `<HostHomeView />`로 교체한다. 다른 분기(hostOccupied / !ctx.state / 진행 중)는 변경 없음.

## 4. 영향 받는 파일 (예상)

| 파일 | 변경 종류 | 비고 |
|---|---|---|
| `web/src/App.tsx` | 수정 | 라우트 1건 추가 |
| `web/src/lib/optionsStorage.ts` | 신규 | localStorage 모듈 |
| `web/src/context/GameContext.tsx` | 수정 | hostOptions/saveHostOptions 노출 |
| `web/src/types/wire.ts` | 수정 | OutgoingMsg에 `host:save-options` 추가 |
| `web/src/views/PublicView/HostHomeView.tsx` | 신규 | 메인 메뉴 |
| `web/src/views/PublicView/HostSettingsView.tsx` | 신규 | 설정 폼 |
| `web/src/views/PublicView/PublicView.tsx` | 수정 | 인라인 폼 제거 + `<HostHomeView />` 사용 |
| `web/src/lib/optionsStorage.test.ts` | 신규 | localStorage 라운드트립 + 페일백 단위 테스트 |
| `web/src/views/PublicView/HostHomeView.test.tsx` | 신규 | 두 버튼 동작 검증(navigate / send 호출 어설션) |
| `web/src/views/PublicView/HostSettingsView.test.tsx` | 신규 | 9 필드 변경 + 저장 → ctx.saveHostOptions 호출 검증 |

## 5. 테스트 계획

| ID | 케이스 | 위치 |
|---|---|---|
| I7-U5-T1 | `loadSavedOptions` — 정상 라운드트립 | optionsStorage.test.ts |
| I7-U5-T2 | `loadSavedOptions` — 잘못된 JSON / 스키마 누락 시 null + 키 삭제 | optionsStorage.test.ts |
| I7-U5-T3 | `loadSavedOptions` — localStorage 비활성 환경 안전 처리 | optionsStorage.test.ts |
| I7-U5-T4 | HostHomeView "게임 시작" → `host:open-room` 송신 | HostHomeView.test.tsx |
| I7-U5-T5 | HostHomeView "설정" → navigate('/public/settings') | HostHomeView.test.tsx |
| I7-U5-T6 | HostSettingsView 9 필드 입력 → "저장 후 메인으로" → saveHostOptions 호출 | HostSettingsView.test.tsx |
| I7-U5-T7 | HostSettingsView 권장값 외 입력 시 경고 노출, 저장 가능 | HostSettingsView.test.tsx |

기존 회귀: 기존 vitest 케이스(45)는 모두 PASS 유지. PublicView 분기 변경은 기존 PlayerView/Public broadcast 테스트와 무관해야 함.

## 6. 비-범위 (Out of Scope)

- 호스트 재접속 시 서버측 옵션 자동 노출(`host:claim` ack 확장) — 다음 이터레이션
- localStorage 키 마이그레이션(v1 → v2) — 본 이터레이션은 v1 단일
- 옵션 프리셋(쉬움/보통/어려움) UI — 범위 외
- 다국어/i18n 프레임워크 — 범위 외

## 7. 사용자 승인 (Approval Gate)

본 Functional Design Patch v1.0을 검토하시고 다음 중 하나로 응답해 주십시오.

- **Continue to Next Stage** — U5 Code Generation으로 진행
- **Request Changes** — 변경 항목을 알려주시면 v1.1로 갱신
