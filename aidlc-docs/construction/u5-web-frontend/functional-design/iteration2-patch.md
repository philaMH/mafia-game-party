# U5 Web Frontend — Functional Design Iteration 2 Patch

**문서 버전**: 1.0
**작성일**: 2026-04-29
**기준 산출물**: U5 v1 functional-design (`information-architecture.md`, `state-model.md`, `interaction-flows.md`)
**상위 변경 명세**: `requirements-iteration2-patch.md` v2.0-patch + `application-design/iteration2-patch.md` v1.0
**처리 방식**: v1 본문 보존, 변경분만 명시.

---

## 1. 변경 요약

| ID | 종류 | 위치 | 변경 |
|---|---|---|---|
| **F-1** | 변경 | `wire.ts` Options | `maxPlayers: number` 필드 추가 |
| **F-2** | 변경 | `wire.ts` IncomingMsg 유니온 | `HostTokenMsg` / `RoomOpenedMsg` / `RoomHostOccupiedMsg` 추가 |
| **F-3** | 변경 | `wire.ts` OutgoingMsg 유니온 | `host:claim` / `host:open-room` / `host:start-room` / `host:terminate-room` / `player:end-self-intro` 추가 |
| **F-4** | 변경 | `reducer.ts` GameState | `hostToken?: string` / `roomOpened: boolean` / `hostOccupied: boolean` 추가 |
| **F-5** | 변경 | `reducer.ts` applyIncoming | 신규 메시지 3종 처리 |
| **F-6** | 변경 | `PublicView.tsx` | 자동 host:claim → 결과 분기 (방 개설 폼 / 차단 화면). 사회자 톤 카피. 본 흐름은 호스트가 게임에 플레이어로 참여하지 않음 |
| **F-7** | 변경 | `PlayerView.tsx` | 방 미개설 게이트 화면 + room:opened 자동 전환 |
| **F-8** | 신규 | PlayerView 자기소개 단계 | "내 자기소개 종료" 버튼 (본인 차례 시) |

---

## 2. 신규 IncomingMsg / OutgoingMsg

```ts
// Incoming
| { type: "host-token"; token: string }
| { type: "room:opened"; options: Options }
| { type: "room:host-occupied" }

// Outgoing
| { type: "host:claim" }
| { type: "host:open-room"; options: Options }
| { type: "host:start-room" }
| { type: "host:terminate-room" }
| { type: "player:end-self-intro" }
```

## 3. State 확장

```ts
interface GameState {
  // ... 기존
  hostToken?: string;       // host-token 수신 시 채움
  roomOpened: boolean;      // room:opened 수신 시 true
  hostOccupied: boolean;    // room:host-occupied 수신 시 true (PublicView 차단 화면)
}
```

## 4. PublicView 흐름

```
WS 연결 성공 → 자동 send("host:claim")
   ↓
host-token 수신 → hostToken 저장 → "방 개설" 폼 표시
                                    (게임 설정 입력: 최대 인원 6~12, 마피아 수 권장 ±1)
   ↓
사용자 제출 → send("host:open-room", options) → roomOpened=true → "참가자를 받습니다"
   ↓
인원 충족 → "게임 시작" 버튼 → send("host:start-room")
   ↓
게임 진행 중 → "강제 종료" 버튼 (HostControls v2)
```

차단 흐름:
```
WS 연결 후 자동 send("host:claim") → room:host-occupied 수신 → "이미 호스트가 운영 중입니다" 차단 화면
```

## 5. PlayerView 흐름

```
WS 연결 → roomOpened===false → "방이 아직 없습니다" 게이트
   ↓ (room:opened 수신)
roomOpened===true → 닉네임 입력 폼 → send("join", name)
   ↓
playerId 수신 → 대기실 진입 → 게임 진행 (v1 동일)
   ↓ (자기소개 단계 + 본인 차례)
"내 자기소개 종료" 버튼 활성 → send("player:end-self-intro")
```

## 6. 단위 테스트

- `reducer.test.ts`: room:opened/host-token/host-occupied 처리 → state 갱신
- `wire.ts` 타입 추가는 컴파일러로 검증 (별도 테스트 불필요)
- 기존 reducer 테스트는 영향 없음 (신규 분기만 추가)

## 7. 커버리지 목표

- 핵심 모듈(reducer.ts) 커버리지 92%+ 유지.
