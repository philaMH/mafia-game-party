# Business Rules — U3 Realtime Transport

**작성일**: 2026-04-26
**문서 버전**: 1.0
**참조**: `domain-entities.md`, `business-logic-model.md`

본 문서는 U3의 사전조건, 가시성 라우팅, 재연결, 송신 큐, 인증, 에러 매핑 규칙을 정리합니다.

---

## 1. Hub 메서드 사전조건 (BR-U3-METHOD)

| 메서드 | 사전조건 | 위반 시 |
|---|---|---|
| `Register` | upgrader.Upgrade 성공한 conn | error 반환 + conn 즉시 close |
| `Unregister` | 어떤 ClientID도 허용 (idempotent) | no-op |
| `Run(ctx)` | 1회 호출 | 두 번째 호출 시 nil 반환 |
| `Close` | 어떤 시점도 허용 (idempotent) | 두 번째 호출 시 nil 반환 |

### 공통 (BR-U3-COMMON)

| ID | 규칙 |
|---|---|
| BR-U3-COMMON-1 | 모든 conn write는 writeLoop goroutine에서만 (gorilla single-writer 요구) |
| BR-U3-COMMON-2 | 모든 메시지는 JSON UTF-8. 바이너리 frame 불사용. |
| BR-U3-COMMON-3 | `type` 필드 누락 또는 미지원 type → `error` 응답 (`code: "VALIDATION_ERROR"`) 후 연결 유지 |
| BR-U3-COMMON-4 | Hub 메서드는 SessionManager 락을 획득하지 않음 — SessionManager 호출은 직접 위임만 |
| BR-U3-COMMON-5 | 패닉 발생 시 `defer recover()` 로 격리 — 해당 클라이언트만 disconnect, Hub 전체는 정상 동작 |

---

## 2. 가시성 라우팅 규칙 (BR-U3-VIS, Q-FD-U3-4=A)

| ID | 규칙 |
|---|---|
| BR-U3-VIS-1 | `VisPublic` 이벤트는 모든 PUBLIC 클라이언트 + 모든 PLAYER 클라이언트(살아있는·사망 포함)에게 송신 (Q-FD-U3-5=A) |
| BR-U3-VIS-2 | `VisPlayer` 이벤트는 `Envelope.PlayerID == Client.PlayerID`인 단일 클라이언트에게만 송신. 해당 PID가 등록되어 있지 않으면 drop |
| BR-U3-VIS-3 | `VisRoleMafia` 이벤트는 SessionManager.Snapshot()의 `Players[*].Alive == true && Role == MAFIA`인 모든 PlayerID의 클라이언트에게만 송신. PUBLIC에는 보내지 않음 |
| BR-U3-VIS-4 | 같은 EventOut의 `Announcement`는 `ForPublicOnly == true`이면 PUBLIC만, `false`(에러 안내)이면 직접 송신자에게만 |
| BR-U3-VIS-5 | 라우팅 실패(대상 0명)는 정상 — drop 후 debug log만 |

---

## 3. 인증 / 권한 규칙 (BR-U3-AUTH, Q-FD-U3-12=B)

| ID | 규칙 |
|---|---|
| BR-U3-AUTH-1 | 호스트 권한 체크는 SessionManager 단독 — Hub는 통과만. 클라이언트가 `host:start` 등을 보내도 Hub는 PlayerID와 HostID 비교 없이 SubmitAction 호출 |
| BR-U3-AUTH-2 | SessionManager가 ErrPermissionDenied 반환 → Hub가 `sendError(c, "PERMISSION_DENIED_ERROR", ...)`로 송신자에게만 안내 |
| BR-U3-AUTH-3 | 토큰은 `resume` 메시지의 token 필드로만 검증. Hub는 토큰을 자체 비교 안 함 — `mgr.ResumePlayer(token)` 호출 결과의 ErrUnknownPlayer로 판단 |
| BR-U3-AUTH-4 | 클라이언트가 `host:create-session`을 보낼 수 있는 시점은 LOBBY가 비었을 때만. 이미 호스트가 있으면 ErrWrongPhase가 반환됨 — Hub는 그대로 전달 |

---

## 4. 재연결 규칙 (BR-U3-RECONNECT, Q-FD-U3-9=A, Q-FD-U3-15=A)

| ID | 규칙 |
|---|---|
| BR-U3-RECONNECT-1 | 같은 PlayerID로 새 클라이언트가 `resume`(또는 `join`) 성공 시, 기존 클라이언트는 즉시 Unregister + WS close (last-connect-wins) |
| BR-U3-RECONNECT-2 | Unregister된 기존 클라이언트의 미송신 메시지는 손실 — 새 클라이언트는 snapshot 메시지로 자기 화면 복구 |
| BR-U3-RECONNECT-3 | `resume` 응답 직후 snapshot 메시지를 1회 push (`type: "snapshot"`). 이후 새 이벤트 stream에 합류 |
| BR-U3-RECONNECT-4 | 토큰 미일치 → ErrUnknownPlayer를 매핑해 `error` 응답 송신 후 연결 유지 (재시도 가능) |
| BR-U3-RECONNECT-5 | `resume`은 단계 무관 허용 (LOBBY/INTRO/NIGHT/DAY/VOTE/RECOUNT/END). U2 BR-U2-TOKEN-4와 일치 |

---

## 5. 송신 큐 / 백프레셔 규칙 (BR-U3-QUEUE, Q-FD-U3-7=A)

| ID | 규칙 |
|---|---|
| BR-U3-QUEUE-1 | 클라이언트당 송신 채널 버퍼 크기는 **16** |
| BR-U3-QUEUE-2 | enqueue가 채널 가득(default branch) → 해당 클라이언트만 Unregister + warn log. SessionManager 호출 흐름은 영향 없음 |
| BR-U3-QUEUE-3 | writeLoop의 `WriteMessage` 실패 → 즉시 conn close + return. readLoop가 ErrClosed 감지하여 Unregister 호출 |
| BR-U3-QUEUE-4 | writeLoop의 ping 송신 실패도 동일 — 즉시 종료 |
| BR-U3-QUEUE-5 | Unregister는 idempotent — close(c.Out)이 이미 닫힌 채널이면 무시 |

### 채널 버퍼 16 근거 (PoC 환경 가정)

| 시나리오 | 메시지 / 초 | 16 버퍼 소화 시간 |
|---|---|---|
| 정상 게임 (Tick 1Hz + Action 빈도) | ~2 msg/s | 8초 |
| 단계 전환 폭증 (예: GameStarted) | ~14 msg | 1회 모두 흡수 |
| 악성 클라이언트 (느린 read) | 가득 → disconnect | 즉각 |

---

## 6. 하트비트 규칙 (BR-U3-HEARTBEAT, Q-FD-U3-8=A)

| ID | 규칙 |
|---|---|
| BR-U3-HEARTBEAT-1 | Read deadline 30초. Pong 수신 시 30초 갱신 |
| BR-U3-HEARTBEAT-2 | 25초마다 서버가 ping 송신 (write goroutine의 pingTicker) |
| BR-U3-HEARTBEAT-3 | Pong 미수신 (read deadline 만료) → readLoop 종료 → Unregister |
| BR-U3-HEARTBEAT-4 | 클라이언트는 별도 ping/pong 구현 불필요 (브라우저 WebSocket이 RFC 6455 표준 frame을 자동 처리) |

---

## 7. 와이어 포맷 규칙 (BR-U3-WIRE)

| ID | 규칙 |
|---|---|
| BR-U3-WIRE-1 | 봉투 평탄 구조 `{type: string, ...payload}` — `data:` 중첩 안 함 (Q-FD-U3-3=A) |
| BR-U3-WIRE-2 | type 명명: kebab-case (`host:start`, `submit:mafia-kill`). Submit 액션은 `submit:` prefix |
| BR-U3-WIRE-3 | `event` 메시지의 페이로드는 `{event: {kind, ...fields}, visibility}` — kind는 PascalCase Go 타입 이름 그대로 |
| BR-U3-WIRE-4 | 시간 필드(deadline 등)는 epoch milliseconds (정수). Go `time.Time` → `t.UnixMilli()` |
| BR-U3-WIRE-5 | 토큰·PlayerID는 server-issued, 클라이언트는 echo만 |
| BR-U3-WIRE-6 | 미지원 type은 `error{code:"VALIDATION_ERROR"}`로 응답 + 연결 유지 (BR-U3-COMMON-3) |
| BR-U3-WIRE-7 | `protocolVersion: "v1"` 필드는 welcome 메시지에 정보용으로만 포함 — 검증 없음 (Q-FD-U3-13=B) |

---

## 8. 에러 매핑 규칙 (BR-U3-ERR)

| ID | 규칙 |
|---|---|
| BR-U3-ERR-1 | SessionManager가 반환한 EngineError를 wire `error{code, message}`로 직접 전달. code는 EngineError.Code 그대로 (예: `VALIDATION_ERROR`, `WRONG_PHASE_ERROR`) |
| BR-U3-ERR-2 | message는 SessionManager가 반환한 EngineError.Message 그대로 (개발자용) — 한국어 안내는 별도 announce 메시지로 송신됨 |
| BR-U3-ERR-3 | JSON 디코딩 에러는 `code: "VALIDATION_ERROR"`, message: `"invalid message"` |
| BR-U3-ERR-4 | SubmitAction 에러 시, `outs[0].Announcement`(announce.RenderError 결과)가 있으면 그 한국어 안내를 announce 메시지로 추가 송신 (BR-U2-ERR-2와 일치) |
| BR-U3-ERR-5 | 에러는 송신자 클라이언트에게만 — 다른 클라이언트에 노출 금지 (BR-U2-ERR-6 복합) |

---

## 9. 메시지 매핑 규칙 (BR-U3-MAP)

incoming type → SessionManager 메서드/Action:

| type | 매핑 |
|---|---|
| `host:create-session` | `mgr.CreateSession(name)` |
| `join` | `mgr.JoinPlayer(name)` |
| `resume` | `mgr.ResumePlayer(token)` |
| `host:start` | `mgr.StartGame(c.PlayerID, options)` |
| `submit:advance-intro` | `mgr.SubmitAction(game.AdvanceIntro{HostID: c.PlayerID})` |
| `submit:mafia-kill` | `mgr.SubmitAction(game.SubmitMafiaKill{Mafia: c.PlayerID, Target: ...})` |
| `submit:doctor-heal` | `mgr.SubmitAction(game.SubmitDoctorHeal{Doctor: c.PlayerID, Target: ...})` |
| `submit:police-check` | `mgr.SubmitAction(game.SubmitPoliceCheck{Police: c.PlayerID, Target: ...})` |
| `submit:end-night` | `mgr.SubmitAction(game.EndNightEarly{HostID: c.PlayerID})` |
| `submit:end-discussion` | `mgr.SubmitAction(game.EndDiscussionEarly{HostID: c.PlayerID})` |
| `submit:vote` | `mgr.SubmitAction(game.SubmitVote{Voter: c.PlayerID, Target: ...})` |
| `host:toggle-voice` | `mgr.SubmitAction(game.ToggleVoice{HostID: c.PlayerID, On: ...})` |
| `host:force-end` | `mgr.SubmitAction(game.ForceEndGame{HostID: c.PlayerID})` |

> 모든 호스트 액션의 `HostID`는 클라이언트의 PlayerID(c.PlayerID)로 채움. SessionManager가 내부 HostID와 비교해 권한 거부.

---

## 10. 로깅 규칙 (BR-U3-LOG, Q-FD-U3-14=A)

| ID | 규칙 |
|---|---|
| BR-U3-LOG-1 | slog 로거 사용. 운영 기본 레벨 INFO. WS 메시지는 DEBUG에서만 |
| BR-U3-LOG-2 | DEBUG 로그는 메시지 type만 기록 — payload는 절대 기록 금지 (토큰/역할 노출) |
| BR-U3-LOG-3 | 연결/해제는 INFO 레벨로 ClientID + Kind + (PLAYER이면) PlayerID |
| BR-U3-LOG-4 | 채널 가득(disconnect) / read timeout / write 실패는 WARN 레벨 |
| BR-U3-LOG-5 | onEvent 내부의 라우팅 실패는 DEBUG 레벨 — 정상 흐름의 일부 |

---

## 11. FR/NFR 추적성

| 출처 | 본 문서 규칙 |
|---|---|
| FR-1.1 (LAN URL + 단일 호스트) | BR-U3-AUTH-1, BR-U3-METHOD Register |
| FR-1.2 (재연결) | BR-U3-RECONNECT-1~5 |
| FR-2.3 (역할 비공개) | BR-U3-VIS-2, BR-U3-VIS-3, BR-U3-LOG-2 |
| FR-7.2 (안내 외부화) | BR-U3-VIS-4 (Announcement만 wire 메시지로 변환) |
| FR-8.4 (안내 풍부) | BR-U3-VIS-4, BR-U3-WIRE-3 |
| NFR-1 (재연결 시 자기 화면 복원) | BR-U3-RECONNECT-3 (snapshot 메시지) |
| NFR-2 (LAN 즉시 반응 + 12명 동접) | BR-U3-QUEUE-* (백프레셔), BR-U3-HEARTBEAT-* |
| NFR-4 (비공개 정보) | BR-U3-VIS-2/3, BR-U3-ERR-5, BR-U3-LOG-2 |
| NFR-7 (외부 서비스 0) | gorilla/websocket 1개 외부 lib만 추가 — 나머지는 표준 lib |

---

## 12. 검증 체크리스트

- [x] Hub 4 메서드 사전조건 명시
- [x] 가시성 3종 라우팅 규칙 명확 (사망 PLAYER 포함)
- [x] 인증은 SessionManager 단독 — Hub는 통과만
- [x] 재연결 5개 규칙 + snapshot 메시지
- [x] 송신 큐 백프레셔 5개 규칙
- [x] 하트비트 25/30초 규칙
- [x] 와이어 포맷 7개 규칙 + protocolVersion 정보용
- [x] 에러 매핑 5개 규칙 — 송신자 한정
- [x] 메시지 매핑 표 14종 incoming
- [x] 로깅 — payload 미기록 (NFR-4 보호)
- [x] 모든 Primary FR/NFR이 규칙으로 매핑됨 (§11)
