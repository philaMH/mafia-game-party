# U5 Code Generation Plan — Iteration 7 (Host Main Menu + Settings Route)

- **버전**: v1.0
- **작성일**: 2026-04-29
- **추적 입력**: `construction/u5-web-frontend/functional-design/iteration7-patch.md` v1.0
- **변경 분류**: Additive(라우트/뷰/모듈) + 분기 정리(PublicView 인라인 폼 제거)

## 진행 체크리스트

### Step A — `optionsStorage.ts` 신규
- [x] A1. `web/src/lib/optionsStorage.ts` 신규.
      - export `loadSavedOptions(): Options | null`
      - export `saveOptions(opts: Options): void`
      - export `clearSavedOptions(): void`
      - 키: `mafia.options.v1`
      - `safeLocalStorage()` 패턴(useToken 참고) 재사용.
      - 파싱 실패/스키마 누락 시 `clearSavedOptions()` 후 `null` 반환.

### Step B — `types/wire.ts` OutgoingMsg variant 추가
- [x] B1. `OutgoingMsg` union에 `| { type: "host:save-options"; options: Options }` 추가 (위치: `host:open-room` 인접).

### Step C — `GameContext.tsx` 확장
- [x] C1. `GameContextValue`에 `hostOptions: Options` + `saveHostOptions(opts: Options): void` 추가.
- [x] C2. `GameProvider` 내부에서 `useState<Options>(() => loadSavedOptions() ?? defaultOptions(8))` 도입.
- [x] C3. `saveHostOptions` 콜백: `setHostOptions` → `saveOptions(opts)` → `send({ type:"host:save-options", options: opts })`.
- [x] C4. `useMemo` value에 두 신규 키 포함.

### Step D — 신규 컴포넌트 `HostHomeView.tsx`
- [x] D1. `web/src/views/PublicView/HostHomeView.tsx` 신규.
      - `useGameContext` + `useNavigate` 사용.
      - 렌더: 노이르 타이틀(`MAFIA`) + "♠ 게임 시작" 버튼 + "⚙ 설정" 버튼.
      - "게임 시작" 클릭 → `ctx.send({ type:"host:open-room", options: ctx.hostOptions })`.
      - "설정" 클릭 → `navigate("/public/settings")`.
      - 기존 PublicView의 `gold-frame`/`btn-noir.primary` 스타일 재사용.

### Step E — 신규 컴포넌트 `HostSettingsView.tsx`
- [x] E1. `web/src/views/PublicView/HostSettingsView.tsx` 신규.
      - `useGameContext` + `useNavigate` 사용.
      - 비-호스트 가드: `useEffect`에서 `!ctx.hostToken || ctx.roomOpened || ctx.hostOccupied` 시 `navigate("/public", { replace: true })`. 가드 충족 전엔 `null` 반환.
      - 9 필드 입력 (숫자 7 + 체크박스 2):
        - maxPlayers, mafiaCount, introSecondsPerPlayer, discussionSeconds, nightMafiaSeconds, nightPoliceSeconds, nightDoctorSeconds, doctorSelfHealAllowed, announcementVoiceOn
        - `useState<Options>(ctx.hostOptions)`로 폼 상태 보유.
      - 권장값 경고: `mafiaCount`가 `defaultOptions(maxPlayers).mafiaCount` 와 1 이상 차이 나면 인라인 경고 노출.
      - 단일 버튼 **"저장 후 메인으로"**: `ctx.saveHostOptions(form)` → `navigate("/public")`.

### Step F — `App.tsx` 라우트 추가
- [x] F1. `<Route path="/public/settings" element={<HostSettingsView />} />` 추가 (`/public` 라우트 인접).

### Step G — `PublicView.tsx` 분기 정리
- [x] G1. 기존 라인 134~189(`ctx.hostToken && !ctx.roomOpened` 분기)의 인라인 폼/`useState<Options>` 제거.
- [x] G2. 같은 분기에서 `<HostHomeView />` 렌더로 교체.
- [x] G3. 사용하지 않게 되는 `import { defaultOptions } from "../../types/wire"` 정리(여전히 다른 곳에서 쓰는지 확인 후).

### Step H — 단위 테스트 신규 3 파일
- [x] H1. `web/src/lib/optionsStorage.test.ts` — T1 라운드트립 / T2 잘못된 JSON → null + 키 삭제 / T3 localStorage 비활성 안전 처리.
- [x] H2. `web/src/views/PublicView/HostHomeView.test.tsx` — T4 "게임 시작" → ctx.send 호출 / T5 "설정" → navigate 호출. `vi.fn()` + 가짜 GameContext 래퍼.
- [x] H3. `web/src/views/PublicView/HostSettingsView.test.tsx` — T6 9 필드 입력 + 저장 → ctx.saveHostOptions 호출 / T7 권장값 외 입력 시 경고 노출 + 저장 가능.

### Step I — 검증
- [x] I1. `npm run typecheck` PASS (또는 `tsc --noEmit`).
- [x] I2. `npm test` PASS, 신규 7 케이스 모두 PASS, 기존 45 케이스 회귀 PASS (총 ≥52 PASS).
- [x] I3. `npm run build` PASS — gzip 회귀 ≤ +3 KB.

### Step J — 산출물
- [x] J1. 코드 변경 요약 audit.md 기록.
- [x] J2. plan 체크박스 모두 [x].
- [x] J3. aidlc-state.md U5 섹션 갱신.

## 변경 파일 목록 (10개)

| 파일 | 종류 | 변경 |
|---|---|---|
| `web/src/lib/optionsStorage.ts` | 신규 | localStorage 모듈 |
| `web/src/lib/optionsStorage.test.ts` | 신규 | 단위 테스트 3 |
| `web/src/types/wire.ts` | 수정 | OutgoingMsg variant 추가 |
| `web/src/context/GameContext.tsx` | 수정 | hostOptions/saveHostOptions 노출 |
| `web/src/App.tsx` | 수정 | `/public/settings` 라우트 |
| `web/src/views/PublicView/HostHomeView.tsx` | 신규 | 메인 메뉴 |
| `web/src/views/PublicView/HostSettingsView.tsx` | 신규 | 설정 폼 |
| `web/src/views/PublicView/PublicView.tsx` | 수정 | 인라인 폼 제거 + HostHomeView 사용 |
| `web/src/views/PublicView/HostHomeView.test.tsx` | 신규 | 통합 테스트 2 |
| `web/src/views/PublicView/HostSettingsView.test.tsx` | 신규 | 통합 테스트 2 |

## 위험·롤백

- **위험**: 라우트 추가 + 분기 정리. 기존 `subscribe-public`/`host:claim` 로직은 PublicView에 그대로 잔존하므로 호스트 진입 시 호스트 토큰 흐름은 변경되지 않음.
- **롤백**: 새 라우트 제거 + 기존 PublicView 인라인 폼 복귀.

## 사용자 승인 (Approval Gate)

본 Code Generation Plan v1.0 검토 후 다음 중 하나로 응답해 주십시오.

- **승인** — 계획대로 코드 생성 시작 (Part 2 실행).
- **수정** — 변경/보완 항목을 알려주시면 v1.1로 갱신.
