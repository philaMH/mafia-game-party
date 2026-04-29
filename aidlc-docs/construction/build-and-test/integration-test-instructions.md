# Integration Test Instructions — mafia-game

**작성일**: 2026-04-26
**대상**: 단위 간 통합 시나리오 (U1+U2+U3+U4+U5 함께 동작)

본 문서는 단일 바이너리를 부팅한 뒤 사람이 손으로 또는 자동화 스크립트로 검증해야 할 통합 시나리오를 정의합니다. 단위별 단위 테스트는 `unit-test-instructions.md`를 참조하세요.

---

## 1. 사전 준비

### 1.1 통합 빌드

```bash
cd web && npm install && npm run build && cd ..
go build -o mafia-game ./cmd/mafia-game
mkdir -p ./data
```

### 1.2 호스트 PC 실행

```bash
./mafia-game --port 8080 --db ./data/mafia.db
```

기대 콘솔:
```
mafia-game listening on:
  http://192.168.1.42:8080      # 또는 localhost:8080 fallback
```

### 1.3 클라이언트 (테스트 머신 / 모바일)

| 라우트 | 사용 |
|---|---|
| `/public` | 호스트 PC 자체에 또는 관전자 모니터 (TTS + 호스트 컨트롤) |
| `/play` | 6명 이상의 플레이어 디바이스 |

---

## 2. 시나리오 매트릭스

| 시나리오 | 단위 | 검증 항목 |
|---|---|---|
| S1: 호스트 부팅 + 기본 라우팅 | U4, U5 | / → /play redirect, /public, /play 모두 200 |
| S2: 6명 게임 1판 (정상 흐름) | U1~U5 | 역할 배정, 자기소개, 야간, 낮, 투표, 종료 |
| S3: 마피아 대표자 동기화 | U1, U3, U5 | MafiaTargetSelected → 다른 마피아 화면 갱신 |
| S4: 의사 자가 보호 토글 | U1, U5 | doctorSelfHealAllowed=true/false 분기 |
| S5: 경찰 한 밤 1회 제한 | U1, U5 | 첫 조사 후 disabled |
| S6: 재연결 (페이지 새로고침) | U2, U3, U5 | resume + snapshot 즉시 복원 |
| S7: 호스트 강제 종료 | U1, U2, U3 | 결과가 game_results에 기록됨 |
| S8: 비정상 종료 후 복원 | U2 | SIGKILL → 재실행 → 자동 복원 |
| S9: TTS 부재 fallback | U5 | window.speechSynthesis 미지원 가짜 환경 → 자막만 |
| S10: /api/results 조회 | U2, U4 | 누적 결과 JSON에 token 미포함 |
| S11: graceful shutdown | U4 | SIGTERM → < 7초 종료 + 데이터 안전 |
| S12: WS 재연결 백오프 | U3, U5 | 서버 재시작 → 클라이언트 자동 재연결 |
| S13: 비공개 정보 라우팅 | U3 | RoleRevealedToPlayer는 본인만, MafiaCohort는 마피아만 |
| S14: 다중 PUBLIC 클라이언트 | U3, U5 | 여러 PublicView가 동일 자막 동시 수신 |
| S15: 12명 동접 한도 | U2 | 13번째 join → ErrValidation |

---

## 3. 핵심 시나리오 상세

### 3.1 S2: 6명 게임 1판 (정상 흐름)

**스텝**:
1. 호스트 PC `/public` 접속 → 닉네임 "host" 입력 → `host:create-session`
2. 5명 플레이어 디바이스가 `/play` 접속 → 닉네임 입력 → `join`
3. 호스트 화면에서 "게임 시작" 클릭
4. `event(GameStarted, RoleRevealedToPlayer × 6, MafiaCohortRevealed, PhaseChanged INTRO)` 수신
5. PUBLIC: "마피아 게임이 시작됩니다…" 자막 + TTS 발화
6. 자기소개 1순위가 PlayerView에 강조됨
7. 호스트 "다음 발언자" 버튼 5번 → 야간 자동 진입 (`PhaseChanged NIGHT`)
8. 마피아 대표자: 살해 대상 선택 / 의사: 보호 / 경찰: 조사
9. 호스트 "야간 마감" → `DeathAnnounced` 또는 `PeacefulNight` 이벤트
10. PUBLIC: 사망자 화면에 ✕ 표시
11. DAY (180초 토론) → 호스트 "토론 조기 종료" 클릭
12. VOTE → 5명이 투표 → `VoteTallied + Eliminated`
13. 마피아 0 또는 ≥ 시민 도달 → `GameEnded`
14. PUBLIC: "시민의 승리" 또는 "마피아의 승리" + Reveal 화면

**검증**:
- ✅ TTS가 한국어로 발화됨 (PUBLIC만)
- ✅ PlayerView는 자기 Role/Keyword만 표시
- ✅ 마피아 viewer는 동료 마피아 닉네임 + 대표자 식별
- ✅ 사망 후 PlayerView는 NIGHT 입력 비활성화
- ✅ 게임 종료 후 `./data/mafia.db`에 결과 1행 추가
  ```bash
  sqlite3 data/mafia.db "SELECT game_id, winner, end_reason FROM game_results ORDER BY ended_at DESC LIMIT 1;"
  ```

### 3.2 S3: 마피아 대표자 동기화

**스텝** (마피아 2명 시나리오):
1. 마피아 A (대표자) 클라이언트에서 살해 대상 P1 선택 → `submit:mafia-kill`
2. 마피아 B 클라이언트에서 `MafiaTargetSelected` 이벤트 수신
3. 마피아 B 화면: "대표자가 선택한 대상: P1" 표시
4. 마피아 B의 PlayerPicker는 disabled (대표자가 아니므로)

**검증**:
- ✅ 마피아 B만 `MafiaTargetSelected` 수신 (시민/의사/경찰 미수신)
- ✅ Public 화면에 송신 안 됨 (NFR-4)

### 3.3 S6: 재연결 (페이지 새로고침)

**스텝**:
1. 플레이어 P1이 마피아로 진행 중 (NIGHT 단계)
2. P1이 브라우저 새로고침 (Cmd+R)
3. 페이지 로드 → `useToken`이 localStorage 토큰 로드
4. WebSocket 재연결 → `welcome` → `resume{token}`
5. 백엔드 `joined{playerId, isHost: false}` + `snapshot{state, your: {role: MAFIA, keyword, mafiaCohort}}`
6. PlayerView 즉시 NIGHT 화면 복원 (자기 역할 + 대상 입력 폼)

**검증**:
- ✅ 1초 내 화면 복구
- ✅ 사용자가 별도 닉네임 재입력 불필요
- ✅ 진행 중 게임 상태 정확

### 3.4 S8: 비정상 종료 후 복원

**스텝**:
1. 게임 진행 중 (DAY 단계, 6명 입장 후 1명 사망)
2. 호스트 PC `kill -9` 강제 종료
3. `./mafia-game --port 8080` 재실행
4. 시작 시 자동 `LoadActiveSnapshot` → 복원
5. 호스트 PC `/public` 재접속 → "이전 게임이 복원되었습니다…" 시스템 안내 표시
6. 모든 플레이어 클라이언트가 자동 재연결 → snapshot으로 복원

**검증**:
- ✅ players[].alive 정확
- ✅ 현재 day, phase, deadline 정확
- ✅ pendingMafiaTarget 등 야간 누적 정확
- ✅ 게임 진행 가능

### 3.5 S11: graceful shutdown

**스텝**:
1. `./mafia-game --port 8080`
2. 게임 시작 후 NIGHT 단계 진입
3. SIGINT (Ctrl+C) 송신
4. 콘솔: `signal received` → `goodbye`
5. < 7초 내 종료
6. 종료 후 `./data/mafia.db`에 `active_snapshot` 1행 존재 (다음 부팅 복원용)

**검증**:
```bash
sqlite3 data/mafia.db "SELECT id, game_id, host_id FROM active_snapshot;"
# 기대: id=1, game_id, host_id 있음
```

### 3.6 S15: 12명 동접 한도

**스텝**:
1. 호스트 1명 + 11명 플레이어 입장 (총 12)
2. 13번째 플레이어가 `join` 시도
3. `error{code: "VALIDATION_ERROR", message: "lobby is full"}` 수신
4. UI: 토스트 "닉네임은 ..." 또는 "lobby is full" 표시 (백엔드 message 직접 표시)

---

## 4. 자동화 스크립트 (선택)

### 4.1 빠른 smoke test

```bash
#!/usr/bin/env bash
set -euo pipefail

./mafia-game --port 18080 &
SERVER=$!
sleep 1

curl -fsS http://localhost:18080/healthz | grep -q ok || { kill $SERVER; exit 1; }
curl -fsS http://localhost:18080/ | head -1 | grep -q "<!doctype html>" || { kill $SERVER; exit 1; }
curl -fsS http://localhost:18080/api/results | grep -q '"results"' || { kill $SERVER; exit 1; }

kill $SERVER
echo "smoke test PASSED"
```

### 4.2 WebSocket 핸드셰이크 (websocat 사용)

```bash
# 별도 터미널
./mafia-game --port 18080

# 다른 터미널
websocat ws://localhost:18080/ws
> {"type":"host:create-session","name":"hostA"}
< {"type":"welcome",...}
< {"type":"joined","playerId":"...","token":"...","isHost":true}
```

---

## 5. 예상 결함 / 환경 의존

| 환경 | 영향 |
|---|---|
| Safari (iOS) | TTS 음성 제한 — `voiceschanged`로 fallback voice 시도 |
| Chrome (Android) | localStorage 정상 |
| 사내 LAN VPN 분리 | LAN IP 검색이 VPN IP를 잡을 수 있음 — 호스트가 직접 확인 |
| 방화벽 (port 8080) | 외부 디바이스 접속 차단 시 룰 추가 필요 |

---

## 6. 검증 체크리스트

- [x] 시나리오 15종 매트릭스
- [x] 핵심 6 시나리오 상세 (S2/S3/S6/S8/S11/S15)
- [x] 자동화 smoke test 스크립트
- [x] 환경 의존 표
