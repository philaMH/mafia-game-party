# U5 Web Frontend · Code Generation Plan — Iteration 10

**Status**: Draft v1.0 — 사용자 승인 대기
**Source**: `aidlc-docs/construction/u5-web-frontend/functional-design/iteration10-patch.md` v1.0 (사용자 승인 2026-04-30T01:20Z)
**Type**: Minimal Patch (신규 hook 1건 + 신규 컴포넌트 1건 + PublicView 통합 + 단위 테스트 4건)

---

## 1. Step 개요

```
Step A — web/src/hooks/useBgm.ts (신규):       단일 Audio element + loop + volume 0.15 + graceful 처리
Step B — web/src/hooks/useBgm.test.ts (신규):  T1~T4 4 케이스
Step C — web/src/views/PublicView/BgmToggle.tsx (신규): 토글 컴포넌트
Step D — web/src/views/PublicView/PublicView.tsx (수정): bgmOn useState + useBgm 호출 + footer 토글 추가
Step E — 검증: typecheck / test / build / go test / go build
Step F — audit.md / aidlc-state.md 동기화
```

---

## 2. Step A — `web/src/hooks/useBgm.ts` 신규

### 2.1 작성할 코드

```ts
import { useEffect, useRef } from "react";

const BGM_SRC = "/audio/bgm.mp3";
const BGM_VOLUME = 0.15;

export interface BgmHandle {
  available: boolean;
}

// useBgm renders a single looping HTMLAudioElement on the host's
// PublicView (Iter10 FR-1~FR-7). Independent from useAudioCueQueue —
// effect cues and BGM coexist without ducking (Q3=B). Graceful: any
// play() rejection or `error` event logs a warning and leaves the game
// untouched. enabled=false pauses without resetting currentTime so a
// subsequent re-enable resumes from the same position.
export function useBgm(enabled: boolean): BgmHandle {
  const available =
    typeof window !== "undefined" && typeof Audio !== "undefined";
  const audioRef = useRef<HTMLAudioElement | null>(null);

  useEffect(() => {
    if (!available) return;
    const el = new Audio(BGM_SRC);
    el.loop = true;
    el.volume = BGM_VOLUME;
    el.preload = "auto";
    const onError = (): void => {
      // eslint-disable-next-line no-console
      console.warn(`[bgm] error event for ${BGM_SRC}`);
    };
    el.addEventListener("error", onError);
    audioRef.current = el;
    return () => {
      el.pause();
      el.removeEventListener("error", onError);
      audioRef.current = null;
    };
  }, [available]);

  useEffect(() => {
    const el = audioRef.current;
    if (!el) return;
    if (enabled) {
      el.play().catch((err) => {
        // eslint-disable-next-line no-console
        console.warn(`[bgm] failed to play ${BGM_SRC}:`, err);
      });
    } else {
      el.pause();
    }
  }, [enabled]);

  return { available };
}
```

### 체크리스트
- [ ] A.1 파일 신규 작성
- [ ] A.2 BGM_SRC / BGM_VOLUME 상수 export 없이 모듈 스코프
- [ ] A.3 unmount cleanup: pause + 리스너 해제 + ref null
- [ ] A.4 enabled effect: play() / pause() 호출, rejection → console.warn

---

## 3. Step B — `web/src/hooks/useBgm.test.ts` 신규

### 3.1 작성할 코드

```ts
import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { useBgm } from "./useBgm";

interface FakeAudio {
  src: string;
  loop: boolean;
  volume: number;
  preload: string;
  paused: boolean;
  currentTime: number;
  play: ReturnType<typeof vi.fn>;
  pause: ReturnType<typeof vi.fn>;
  addEventListener: ReturnType<typeof vi.fn>;
  removeEventListener: ReturnType<typeof vi.fn>;
}

let lastAudio: FakeAudio | null = null;
let playImpl: () => Promise<void> = () => Promise.resolve();
const originalAudio = globalThis.Audio;

beforeEach(() => {
  lastAudio = null;
  playImpl = () => Promise.resolve();
  // @ts-expect-error — installing a fake Audio constructor for the test
  globalThis.Audio = vi.fn().mockImplementation((src?: string) => {
    const fake: FakeAudio = {
      src: src ?? "",
      loop: false,
      volume: 1,
      preload: "",
      paused: true,
      currentTime: 0,
      play: vi.fn(() => {
        fake.paused = false;
        return playImpl();
      }),
      pause: vi.fn(() => {
        fake.paused = true;
      }),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    };
    lastAudio = fake;
    return fake;
  });
});

afterEach(() => {
  globalThis.Audio = originalAudio;
  vi.restoreAllMocks();
});

describe("useBgm", () => {
  // T1 — enabled=true 마운트 시 loop / volume / play() 호출
  it("creates a looping element at volume 0.15 and plays when enabled", async () => {
    renderHook(() => useBgm(true));
    await act(async () => {
      await Promise.resolve();
    });
    expect(lastAudio).not.toBeNull();
    expect(lastAudio!.src).toContain("/audio/bgm.mp3");
    expect(lastAudio!.loop).toBe(true);
    expect(lastAudio!.volume).toBe(0.15);
    expect(lastAudio!.play).toHaveBeenCalledTimes(1);
  });

  // T2 — enabled toggle false → pause(), currentTime 보존
  it("pauses without resetting currentTime when toggled off", async () => {
    const { rerender } = renderHook(({ on }) => useBgm(on), {
      initialProps: { on: true },
    });
    await act(async () => {
      await Promise.resolve();
    });
    lastAudio!.currentTime = 12.34;
    rerender({ on: false });
    await act(async () => {
      await Promise.resolve();
    });
    expect(lastAudio!.pause).toHaveBeenCalled();
    expect(lastAudio!.currentTime).toBe(12.34);
  });

  // T3 — play() rejection 시 console.warn 만, 게임 진행 무중단
  it("logs a warning when play() rejects (graceful)", async () => {
    const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
    playImpl = () => Promise.reject(new Error("autoplay denied"));
    renderHook(() => useBgm(true));
    await act(async () => {
      await Promise.resolve();
      await Promise.resolve();
    });
    expect(warn).toHaveBeenCalled();
    const arg = warn.mock.calls[0]?.[0];
    expect(String(arg)).toContain("[bgm] failed to play");
  });

  // T4 — unmount 시 pause + 리스너 해제 + ref null
  it("pauses and removes listener on unmount", async () => {
    const { unmount } = renderHook(() => useBgm(true));
    await act(async () => {
      await Promise.resolve();
    });
    const el = lastAudio!;
    unmount();
    expect(el.pause).toHaveBeenCalled();
    expect(el.removeEventListener).toHaveBeenCalledWith(
      "error",
      expect.any(Function),
    );
  });
});
```

### 체크리스트
- [ ] B.1 FakeAudio + globalThis.Audio mock 셋업
- [ ] B.2 T1 loop/volume/play 검증
- [ ] B.3 T2 토글 OFF currentTime 보존 검증
- [ ] B.4 T3 play rejection graceful 검증
- [ ] B.5 T4 unmount cleanup 검증

---

## 4. Step C — `web/src/views/PublicView/BgmToggle.tsx` 신규

### 4.1 작성할 코드

```tsx
interface Props {
  on: boolean;
  onChange: (on: boolean) => void;
}

export function BgmToggle({ on, onChange }: Props) {
  return (
    <button
      type="button"
      className={"btn-noir sm" + (on ? "" : " ghost")}
      onClick={() => onChange(!on)}
      style={{
        borderColor: on ? "var(--gold)" : undefined,
        color: on ? "var(--gold)" : undefined,
      }}
    >
      🎵 배경음 {on ? "ON" : "OFF"}
    </button>
  );
}
```

### 체크리스트
- [ ] C.1 파일 신규 작성
- [ ] C.2 VoiceToggle 와 동일한 `btn-noir sm` 스타일
- [ ] C.3 ON 시 `var(--gold)` 색상 (VoiceToggle 의 `--alive` 와 시각 구분)

---

## 5. Step D — `web/src/views/PublicView/PublicView.tsx` 수정

### 5.1 변경 표

| 위치 | 변경 |
|------|------|
| import 추가 | `import { useBgm } from "../../hooks/useBgm";` / `import { BgmToggle } from "./BgmToggle";` |
| `PublicView()` 본문 | `const [bgmOn, setBgmOn] = useState(true);` 추가 (기존 `claimSent` useState 옆) |
| useBgm 호출 | `useBgm(isHost && bgmOn && Boolean(ctx.hostToken));` 추가 (`isHost` 변수 정의 직후) |
| footer | `{isHost && <BgmToggle on={bgmOn} onChange={setBgmOn} />}` 를 `{isHost && <VoiceToggle ... />}` 직전 또는 직후에 삽입 |

### 5.2 patch 의도

- `bgmOn` 초기값 `true` — Q1=A FR-1 priming 직후 자동 시작
- `Boolean(ctx.hostToken)` 가드 — priming 통과 신호 (FR-1 / NFR-5 autoplay 정책)
- `useBgm` 입력에 `paused` / `phase==="END"` 미포함 — FR-4 게임 상태와 무관하게 유지

### 체크리스트
- [ ] D.1 import 2건 추가
- [ ] D.2 `bgmOn` useState 추가
- [ ] D.3 `useBgm(...)` 호출 추가 (`isHost` 파생 변수 직후)
- [ ] D.4 footer `BgmToggle` 렌더 추가 (`VoiceToggle` 옆)

---

## 6. Step E — 검증

```bash
cd /Users/myunghoonkang/study/saltware-ai-dlc/mafia-game/.claude/worktrees/feature+bgm/web

npm run typecheck
npm test               # 71 → ≥ 75 PASS 목표 (T1~T4 추가)
npm run build          # JS gzip ≤ 66.21 KB 목표

cd /Users/myunghoonkang/study/saltware-ai-dlc/mafia-game/.claude/worktrees/feature+bgm
go test ./... -count=1 -race    # 6 패키지 PASS
go build -o /tmp/mafia-game-iter10 ./cmd/mafia-game
```

### 체크리스트
- [ ] E.1 `npm run typecheck` PASS
- [ ] E.2 `npm test` ≥ 75 PASS (이전 71 + 신규 4)
- [ ] E.3 `npm run build` 성공, JS gzip ≤ 66.21 KB
- [ ] E.4 `go test ./... -race` 6 패키지 PASS (회귀 무영향)
- [ ] E.5 `go build` 성공

---

## 7. Step F — 동기화

- `aidlc-docs/audit.md` 에 본 plan 승인 + 코드 생성 결과 + 검증 결과 entry 추가
- `aidlc-docs/aidlc-state.md` U5 체크박스 갱신 (Functional Design Patch [x] / Code Generation [x])
- 변경 파일 목록 / 커버리지 / gzip 차이 / npm test 카운트 명시

### 체크리스트
- [ ] F.1 audit.md append
- [ ] F.2 aidlc-state.md update

---

## 8. DoD (Code Generation Plan)

- [x] 4 Step (A~D) 코드 명세 작성
- [x] 검증 명령(E) + 동기화(F) 정의
- [x] FR-1~FR-7 매핑이 Functional Design Patch §5 와 일치
- [ ] 사용자 승인
