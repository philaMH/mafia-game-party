# Business Rules — U2 Session, Persistence & Announce

**작성일**: 2026-04-26
**문서 버전**: 1.0
**참조**: `domain-entities.md`, `business-logic-model.md`

본 문서는 U2의 사전조건, 권한·락·영속화 정책, 안내 매핑 규칙, 트랜잭션 경계, 에러 매핑 규칙을 정리합니다.

---

## 1. SessionManager 메서드 사전조건

### 1.1 공통 (BR-COMMON)

| ID | 규칙 |
|---|---|
| BR-U2-COMMON-1 | 모든 공개 메서드는 진입 시 `s.mu.Lock()`, 종료 시 `s.mu.Unlock()` 보장 |
| BR-U2-COMMON-2 | 호출자 컨텍스트 cancellation은 단일 메서드 단위로 존중 (장기 작업 없음 — 모든 호출은 < 50ms 가정) |
| BR-U2-COMMON-3 | 메서드 실행 중 panic은 회복하지 않음 — 호스트 PC 재시작 + 자동 복원에 의존 (NFR-1) |

### 1.2 메서드별 (BR-U2-METHOD)

| 메서드 | 사전조건 | 위반 시 에러 |
|---|---|---|
| `CreateSession` | `Started=false`, `LoadActiveSnapshot=found:false` | `ErrWrongPhase`, `ErrValidation` |
| `JoinPlayer` | `Started=false`, `len(Members) < 12`, name 중복 없음 | `ErrWrongPhase`, `ErrValidation` |
| `ResumePlayer` | token 매칭되는 Member 존재 | `ErrUnknownPlayer` |
| `StartGame` | `hostID == HostID`, `Started=false`, `len(Members) ≥ 6` | `ErrPermissionDenied`, `ErrWrongPhase`, `ErrValidation` |
| `SubmitAction` | `Started=true` | `ErrWrongPhase`. 그 외 위반은 Engine.Apply에서 처리 |
| `Tick` | `Started=true`이면 처리, 아니면 no-op | (반환값 없음) |
| `Subscribe` | 항상 허용 | (없음) |
| `Close` | 1회만 호출 가능 | 두 번째 호출 시 no-op |

---

## 2. 단일 GM 락 정책 (BR-U2-LOCK)

| ID | 규칙 |
|---|---|
| BR-U2-LOCK-1 | 단일 `sync.Mutex`로 SessionManager의 모든 공개 메서드와 `tickLoop`가 직렬화됨 (Q-FD-U2-1=A) |
| BR-U2-LOCK-2 | Engine의 모든 메서드 호출은 락 안에서만 (`Engine.Apply`, `Engine.Tick`, `Engine.Snapshot`, `Engine.Restore`) — Engine 자체 단일 스레드 가정 (NFR-U1-C1) |
| BR-U2-LOCK-3 | PersistenceStore 호출도 락 안에서 동기 — 다만 Close()만 락 외부에서 호출 가능 |
| BR-U2-LOCK-4 | Subscribe 등록자(EventHandler) 콜백은 **락 안에서** 호출됨 — 핸들러는 빠르게 반환, 무거운 작업은 별도 고루틴으로 |
| BR-U2-LOCK-5 | tickLoop 고루틴이 멈춰있는 경우(예: handler에서 deadlock) 1초 ticker는 누적 — Tick은 멱등이므로 catch-up 시 안전 |

---

## 3. 영속화 정책 (BR-U2-PERSIST)

| ID | 규칙 | 출처 |
|---|---|---|
| BR-U2-PERSIST-1 | `PhaseChanged` 이벤트 발생 시 동기 SaveSnapshot | Q-FD-U2-2=A |
| BR-U2-PERSIST-2 | `DeathAnnounced`, `Eliminated`, `MafiaRepresentativeReassigned` 이벤트 발생 시도 동기 SaveSnapshot | Q-FD-U2-2=A |
| BR-U2-PERSIST-3 | `GameEnded` 이벤트 발생 시: ① SaveResult INSERT ② DeleteActiveSnapshot ③ `Started=false`로 마킹 (한 트랜잭션) | NFR-1 |
| BR-U2-PERSIST-4 | LOBBY 단계는 active_snapshot 미저장 (게임 시작 전) | 단순화 |
| BR-U2-PERSIST-5 | SaveSnapshot 실패 시 호스트에게 ERROR 안내 발행, 게임 진행은 계속 (메모리 상에 state 보존) — 다음 PhaseChanged에서 재시도 |
| BR-U2-PERSIST-6 | events 테이블 기록은 옵션. 기본값 OFF (게임 1판 후 디버그가 필요 없으면 비활성). `SessionOpts.EventLog=true`로 활성화 |
| BR-U2-PERSIST-7 | SQLite 모드: `journal_mode=WAL`, `synchronous=NORMAL` (NFR-1과 성능 균형) |
| BR-U2-PERSIST-8 | DB 파일 경로 우선순위: ① `MAFIA_DB_PATH` 환경변수, ② CLI 플래그, ③ 기본 `./data/mafia.db` (Q-FD-U2-10=A) |

### 트랜잭션 경계

| 작업 | 트랜잭션 |
|---|---|
| SaveSnapshot | 단일 INSERT OR REPLACE — 자동 트랜잭션 |
| SaveResult + DeleteActiveSnapshot | `BEGIN; INSERT; DELETE; COMMIT;` (원자성 보장) |
| AppendEvent | 단일 INSERT |

---

## 4. 자동 복원 정책 (BR-U2-RESTORE)

| ID | 규칙 | 출처 |
|---|---|---|
| BR-U2-RESTORE-1 | NewSessionManager 생성 시 LoadActiveSnapshot 호출 | Q-FD-U2-3=A |
| BR-U2-RESTORE-2 | 발견된 스냅샷이 있으면 Engine.Restore 호출 + Members 복원 | Q-FD-U2-3=A |
| BR-U2-RESTORE-3 | Restore 실패 시 손상 스냅샷 별도 보관(archive 테이블 또는 파일) + 새 LOBBY로 시작 | 견고성 |
| BR-U2-RESTORE-4 | 복원된 세션은 `Started = (state.Phase != LOBBY && state.Phase != END)` |
| BR-U2-RESTORE-5 | 복원 직후 호스트 화면에 "이전 게임이 복원되었습니다…" 시스템 안내 발행 (PublicView 토스트) — U3에 의해 호스트에게 전달 |
| BR-U2-RESTORE-6 | 복원된 게임이 PhaseEnd 상태이면 자동으로 SaveResult + DeleteActiveSnapshot 처리 후 LOBBY 진입 (정상 종료 직후 PC가 재시작된 경우 대비) |

---

## 5. 토큰 / 재연결 정책 (BR-U2-TOKEN)

| ID | 규칙 | 출처 |
|---|---|---|
| BR-U2-TOKEN-1 | JoinPlayer 시 32바이트(64 hex char) `crypto/rand` 토큰 발급 | Q-FD-U2-4=A |
| BR-U2-TOKEN-2 | 같은 게임 내 토큰 중복 차단 (재발급) |
| BR-U2-TOKEN-3 | 토큰은 게임 1판 동안 불변. GameEnded 후 다음 LOBBY 진입 시 신규 발급 |
| BR-U2-TOKEN-4 | ResumePlayer는 단계 무관 허용 (LOBBY/INTRO/NIGHT/DAY/VOTE/RECOUNT/END 모두) |
| BR-U2-TOKEN-5 | 토큰 미일치 시 `ErrUnknownPlayer` |
| BR-U2-TOKEN-6 | ResumePlayer 후 PrivateView 빌드해서 반환 — 본인 Role/Keyword 노출 |

---

## 6. JoinPlayer 정책 (BR-U2-JOIN)

| ID | 규칙 | 출처 |
|---|---|---|
| BR-U2-JOIN-1 | LOBBY에서만 신규 입장 허용 | Q-FD-U2-11=A |
| BR-U2-JOIN-2 | 인원 ≤ 12 (FR-1.3 상한) |
| BR-U2-JOIN-3 | 같은 닉네임 중복 차단 (`ErrValidation`) |
| BR-U2-JOIN-4 | 호스트는 CreateSession에서 자동 입장됨 — 별도 JoinPlayer 불필요 |
| BR-U2-JOIN-5 | StartGame 시 `len(Members) ≥ 6` 검증 — 미만이면 거부 |

---

## 7. 안내 매핑 규칙 (BR-U2-CATALOG)

| ID | 규칙 | 출처 |
|---|---|---|
| BR-U2-CAT-1 | 비공개 이벤트(RoleRevealedToPlayer, MafiaCohortRevealed, MafiaTargetSelected, PoliceResult, MafiaRepresentativeReassigned)는 안내 미발행 — 클라이언트(U5)가 자체 표시 | NFR-4 비공개 보호 |
| BR-U2-CAT-2 | 공개 이벤트는 모두 한국어 안내 발행 (FR-8.4 풍부) — `domain-entities.md` §5 표 |
| BR-U2-CAT-3 | 모든 안내는 한국어 (FR-8.3) — 다국어 미지원 (Non-Goal) |
| BR-U2-CAT-4 | 톤은 근엄·차분·고전적 진행자 (Q-FD-U2-8=A) |
| BR-U2-CAT-5 | `VoteTallied{Eliminated≠nil, Recount=false}` 단일 최다 케이스는 무음 — 직후 발행되는 `Eliminated` 이벤트가 안내 발행 |
| BR-U2-CAT-6 | 보간 변수: `{name}` (Members[id].Name), `{victim}`, `{day}` (state.Day), `{role_kr}` |
| BR-U2-CAT-7 | `Subtitle == Speech` 동일 (단순화) — 향후 미세조정 시 분리 가능 |
| BR-U2-CAT-8 | `ForPublicOnly=true`이면 PlayerView에 송신 안 함 (모든 카탈로그 항목이 `ForPublicOnly=true`) |
| BR-U2-CAT-9 | 카탈로그는 `AnnouncementCatalog` 인터페이스로 추상화 → FR-7.2 외부화 가능 (현재 `defaultCatalog` Go 구현체 1개) |

---

## 8. 에러 매핑 규칙 (BR-U2-ERR)

| ID | 규칙 | 출처 |
|---|---|---|
| BR-U2-ERR-1 | EngineError 9종 모두 한국어 사용자 메시지 매핑 (`domain-entities.md` §5 ErrorAnnouncement 표) | Q-FD-U2-6=A |
| BR-U2-ERR-2 | 매핑은 U2가 수행 — 백엔드가 안내를 만들어 WS 응답으로 전송 |
| BR-U2-ERR-3 | `ValidationErrors` (다중 위반)는 줄바꿈 또는 bullet으로 결합 후 단일 안내로 전송 |
| BR-U2-ERR-4 | 권한/단계/사망 에러는 토스트로 (PlayerView 우상단), 검증 에러는 폼 옆 inline |
| BR-U2-ERR-5 | 에러 발생 시 state 변경 없음 (Engine 보장, NFR-U1-R2) — 영속화도 없음 |
| BR-U2-ERR-6 | 에러 안내는 송신자(action 발신자)의 PlayerView에만 전송 — 다른 화면에 노출 안 함 (NFR-4 비공개) |

---

## 9. 가시성 / 마스킹 규칙 (BR-U2-MASK)

| ID | 규칙 |
|---|---|
| BR-U2-MASK-1 | PublicView로 송신되는 모든 State는 Players[*].Role / Keyword 빈 문자열로 마스킹 |
| BR-U2-MASK-2 | PlayerView로 송신될 때, viewer 본인의 Role/Keyword만 노출. 다른 플레이어는 마스킹 |
| BR-U2-MASK-3 | 마피아 viewer에게는 다른 마피아의 Role도 노출 (Keyword는 같으므로 본인 것과 동일) |
| BR-U2-MASK-4 | `Phase == END` 도달 시 모든 시청자에게 모든 플레이어 Role 공개 (Reveal) |
| BR-U2-MASK-5 | 마스킹 책임은 U2 (PrivateView 빌더). U3는 라우팅만, U5는 표시만 |

---

## 10. 백그라운드 ticker 정책 (BR-U2-TICK)

| ID | 규칙 | 출처 |
|---|---|---|
| BR-U2-TICK-1 | SessionManager가 1초 간격 `time.Ticker` 보유 (Q-FD-U2-5=A) |
| BR-U2-TICK-2 | Tick 호출은 `Started=true`일 때만 Engine.Tick 호출 |
| BR-U2-TICK-3 | tickLoop는 `stopCh`로 graceful 종료 |
| BR-U2-TICK-4 | Tick 호출이 락을 1초보다 오래 유지하면 다음 ticker 신호는 채널 버퍼(1)에 의해 1회 보존됨 — 그 이상은 누락 (Engine.Tick 멱등성으로 안전) |

---

## 11. graceful shutdown 규칙 (BR-U2-CLOSE)

| ID | 규칙 |
|---|---|
| BR-U2-CLOSE-1 | Close는 stopCh 닫고 락 획득 후 마지막 SaveSnapshot 실행 |
| BR-U2-CLOSE-2 | 진행 중 게임이면 active_snapshot 보존 (다음 부팅 자동 복원) |
| BR-U2-CLOSE-3 | LOBBY 단계라면 SaveSnapshot 안 함 (BR-U2-PERSIST-4) |
| BR-U2-CLOSE-4 | persistence.Close()로 SQLite 핸들 정리 |
| BR-U2-CLOSE-5 | Close 두 번 호출 안전 — 두 번째 호출은 no-op |

---

## 12. FR/NFR 추적성

| 출처 | 본 문서 규칙 |
|---|---|
| FR-1.1 (단일 호스트 PC 1세션) | BR-U2-METHOD CreateSession, BR-U2-JOIN-1 |
| FR-1.2 (닉네임 + 재연결) | BR-U2-JOIN-3, BR-U2-TOKEN-* |
| FR-4.3 (마피아 대표자 권한) | Engine 위임 — U1 BR-REP-3 + U2 ensureMember 보조 |
| FR-6.1 (결과 누적 저장) | BR-U2-PERSIST-3 |
| FR-6.2 (스냅샷 영속화) | BR-U2-PERSIST-1, BR-U2-PERSIST-2 |
| FR-6.3 (결과 조회) | BR-U2-PERSIST + ListResults |
| FR-7.2 (안내 외부화 인터페이스) | BR-U2-CAT-9 |
| FR-8.4 (풍부 안내) | BR-U2-CAT-2 + 카탈로그 25개 |
| FR-8.3 (한국어 한정, 근엄 톤) | BR-U2-CAT-3, BR-U2-CAT-4 |
| NFR-1 (안정성·복원) | BR-U2-PERSIST-*, BR-U2-RESTORE-*, BR-U2-CLOSE-* |
| NFR-4 (비공개 정보 라우팅) | BR-U2-MASK-*, BR-U2-CAT-1 |
| NFR-7 (외부 서비스 0) | BR-U2-PERSIST-7 (단일 SQLite 파일) |

---

## 13. 검증 체크리스트

- [x] 모든 SessionManager 메서드의 사전조건 명시
- [x] 단일 GM 락 5개 규칙
- [x] 영속화 트리거 정확히 식별 (PhaseChanged + 사망 이벤트)
- [x] 자동 복원 6개 규칙 + 손상 스냅샷 처리
- [x] 토큰 발급/검증 6개 규칙
- [x] 카탈로그 매핑 9개 규칙 + 비공개 이벤트 미발행
- [x] 에러 매핑 6개 규칙
- [x] 마스킹 5개 규칙
- [x] tickLoop 4개 규칙
- [x] graceful shutdown 5개 규칙
- [x] 모든 Primary FR/NFR이 규칙으로 매핑됨 (§12)
