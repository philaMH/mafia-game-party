# U5 Web Frontend · Functional Design Patch — Iteration 10

**Status**: Draft v1.0 — 사용자 승인 대기
**Source**: `aidlc-docs/inception/requirements/iteration10-bgm-requirements.md` v1.0 (사용자 승인 2026-04-30T01:00Z)
**Plan**: `aidlc-docs/construction/plans/iteration10-execution-plan.md` v1.0 (사용자 승인 2026-04-30T01:10Z)
**Type**: Minimal Patch (신규 hook 1건 + 신규 컴포넌트 1건 + PublicView 통합)

---

## 1. 변경 의도

호스트 화면(`PublicView`, `isHost === true`) 에서 단일 BGM 음원(`/audio/bgm.mp3`) 을 무한 루프 재생한다. 기존 효과음 재생 채널(`useAudioCueQueue`) 과는 두 개의 독립된 `HTMLAudioElement` 로 분리해 운용하므로, 효과음 cue 와 BGM 이 동시에 재생되어도 서로 간섭하지 않는다. 덕킹은 적용하지 않는다.

서버측(U1/U2/U3/U4) 변경 없음. wire protocol / 카탈로그 / `/audio/` 라우팅 모두 보존.

---

## 2. 신규 hook — `web/src/hooks/useBgm.ts`

### 2.1 인터페이스
```ts
export interface BgmHandle {
  available: boolean;       // typeof Audio !== "undefined"
}

export function useBgm(enabled: boolean): BgmHandle;
```

### 2.2 동작 표

| 단계 | 트리거 | 동작 |
|------|--------|------|
| 마운트 | `useEffect` 최초 진입 | `available` 평가, 단일 `HTMLAudioElement` 생성, `src = "/audio/bgm.mp3"`, `loop = true`, `volume = 0.15`, `preload = "auto"` |
| enabled true (마운트 또는 토글 ON) | `useEffect([enabled])` | `audio.play()` 호출. rejection 시 `console.warn("[bgm] failed to play /audio/bgm.mp3", err)` 만 출력 (FR-5) |
| enabled false (토글 OFF) | `useEffect([enabled])` | `audio.pause()` 만 호출, `currentTime` 보존 (FR-2 — 다음 ON 시 같은 위치에서 재개) |
| `error` 이벤트 | audio 엘리먼트 자체에서 발생 | `console.warn("[bgm] error event for /audio/bgm.mp3")` (FR-5) |
| 언마운트 | `useEffect` cleanup | `audio.pause()`, error 리스너 제거, ref null (FR-7 — 메모리 누수 방지) |

### 2.3 상태 다이어그램

```
            +----------+   enabled=true     +----------+
            |  IDLE    |  ----------------> | PLAYING  |
            | (paused) |  <---------------- | (looping)|
            +----------+   enabled=false    +----------+
                  ^                              |
                  |                              | error / play() reject
                  |  console.warn 만 출력         |
                  +------------------------------+
                  (게임 진행 무중단 — FR-5)
```

### 2.4 보존 / 가드

- `available === false` (Audio 글로벌 부재) 시 hook 은 no-op. `enabled` 변동에도 아무 호출하지 않음.
- 재생 중 `enabled` true → false → true 토글 시 `currentTime` 보존되어 끊김 없이 재개.
- 동일 컴포넌트 트리에서 중복 마운트 발생해도 단일 ref 로 단 1개 element 만 유지.

---

## 3. 신규 컴포넌트 — `web/src/views/PublicView/BgmToggle.tsx`

### 3.1 Props
```ts
interface Props {
  on: boolean;
  onChange(on: boolean): void;
}
```

### 3.2 렌더 명세
- 클래스: `btn-noir sm` (VoiceToggle 와 동일 스타일 계열)
- 라벨: `🎵 배경음 ON` / `🎵 배경음 OFF`
- borderColor / color: `var(--gold)` (VoiceToggle 의 `--alive` 와 시각적으로 구분)
- disabled 조건: 없음 (Audio 글로벌은 useBgm 내부에서 가드)
- onClick: `onChange(!on)`

### 3.3 배치
`PublicView.tsx` footer 영역, `<VoiceToggle ... />` 직전(또는 직후) 에 호스트 한정 노출.

```jsx
{isHost && <BgmToggle on={bgmOn} onChange={setBgmOn} />}
{isHost && <VoiceToggle ... />}
```

---

## 4. `PublicView.tsx` 수정 표

| 항목 | 종류 | 위치 | 동작 |
|------|------|------|------|
| `bgmOn` useState | 신설 | 컴포넌트 본문 | 초기값 `true` (FR-1 — priming 완료 시 자동 시작) |
| `useBgm(...)` 호출 | 신설 | `claimSent` useEffect 다음 | `useBgm(isHost && bgmOn && Boolean(ctx.hostToken))` — hostToken 보유 == 호스트 priming 통과 |
| `BgmToggle` import + 렌더 | 신설 | footer 영역 | 호스트 한정 노출 |

**가드 근거**: `ctx.hostToken` 은 `room:opened` / `host:claim` 이후 채워지며, 이 흐름은 사용자가 메인 메뉴에서 "방 개설" 버튼을 눌러 priming 한 결과로 진입한다. 따라서 `Boolean(ctx.hostToken)` 가 곧 priming 통과 신호로 충분하다 (NFR-5 autoplay 정책 준수).

---

## 5. 요구사항 추적

| 요구사항 | 구현 위치 |
|----------|-----------|
| FR-1 priming 직후 자동 시작 | `useBgm(isHost && bgmOn && hostToken)` |
| FR-2 별도 토글 | `BgmToggle.tsx` + `bgmOn` useState |
| FR-3 저볼륨 + 덕킹 없음 | `audio.volume = 0.15`, 효과음 큐와 독립 element |
| FR-4 Pause/GameEnded 유지 | hook 입력에 `paused`/`phase==="END"` 미사용 — 게임 상태와 분리 |
| FR-5 graceful 실패 | `play()` rejection / error 리스너 → `console.warn` |
| FR-6 호스트 한정 | `isHost &&` 가드, `BgmToggle` 도 `isHost &&` 가드 |
| FR-7 cleanup | unmount 시 `pause()` + 리스너 제거 + ref null |

---

## 6. 회귀 영향

| 대상 | 영향 |
|------|------|
| `useAudioCueQueue` | 무영향 — 별도 Audio element, 분리된 hook |
| `VoiceToggle` | 무영향 — 컴포넌트 변경 없음 |
| `GameContext` | 무영향 — reducer/state 변경 없음 |
| Go 서버 (U1~U4) | 무영향 |
| 빌드 사이즈 | 약 +0.3~0.4 KB gzip 예상 (NFR-1 ≤ +0.5 KB 이내) |

---

## 7. DoD (Functional Design Patch)

- [x] 신규 hook 인터페이스/동작 표/상태 다이어그램 명세
- [x] 신규 컴포넌트 props/스타일/배치 명세
- [x] PublicView 통합 위치/가드 명세
- [x] FR-1~FR-7 추적 완료
- [x] 회귀 영향 분석 완료
- [ ] 사용자 승인
