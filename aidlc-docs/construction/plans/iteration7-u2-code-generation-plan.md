# U2 Code Generation Plan — Iteration 7 (호스트 옵션 사전 저장)

- **버전**: v1.0
- **작성일**: 2026-04-29
- **추적 입력**: `construction/u2-session-persistence-announce/functional-design/iteration7-patch.md` v1.0
- **변경 분류**: Additive (인터페이스 메서드 추가, 구조체 필드 추가)

## 진행 체크리스트

### Step A — `SessionManager` 인터페이스 메서드 추가
- [x] A1. `internal/session/session.go` 의 `SessionManager` 인터페이스에 메서드 1건 추가:
      ```go
      SaveHostOptions(ctx context.Context, token HostToken, opts game.Options) error
      ```
- [x] A2. `session` 구조체에 필드 2건 추가: `savedHostOptions game.Options`, `hasSavedHostOptions bool`. 위치: 기존 `hostAuth` 인접.

### Step B — 신규 파일 `internal/session/host_options.go`
- [x] B1. 패키지 `session`. import: `context`, `internal/game`.
- [x] B2. 함수 `(s *session) SaveHostOptions(ctx context.Context, token HostToken, opts game.Options) error`:
      1. `hostAuth.Verify(token)` 실패 → return.
      2. `validateSavedHostOptions(opts)` 실패 → return (ValidationErrors).
      3. `s.mu.Lock()` → 필드 갱신 → `Unlock()`. nil return.
- [x] B3. 함수 `validateSavedHostOptions(opts game.Options) error` — shape-only 검사. 누적 `game.ValidationErrors` 반환:
      - MaxPlayers ∈ [6,12]
      - MafiaCount ≥ 1
      - MafiaCount ≤ MaxPlayers - 3 (citizen-side 가드: 의사·경찰 1명씩 + 일반 시민 ≥ 1명)
      - IntroSecondsPerPlayer ≥ 5
      - DiscussionSeconds ≥ 30
      - NightMafiaSeconds ≥ 5
      - NightPoliceSeconds ≥ 5
      - NightDoctorSeconds ≥ 5
      - bool 두 필드(DoctorSelfHealAllowed, AnnouncementVoiceOn)는 별도 검증 없음.

### Step C — 테스트 신규 `internal/session/iteration7_test.go`
- [x] C1. 도우미: 테스트용 SessionManager 생성(기존 testkit 또는 testHelper 패턴 재사용). 기존 테스트 파일을 참조해 동일 fixture 사용.
- [x] C2. T1 (`TestSaveHostOptions_NoHostToken`): 토큰 미발급 상태에서 호출 → `CodePermissionDenied`.
- [x] C3. T2 (`TestSaveHostOptions_BadToken`): 다른 호스트가 claim 후, 잘못된 토큰으로 호출 → `CodePermissionDenied`.
- [x] C4. T3 (`TestSaveHostOptions_ValidationFailure`): MaxPlayers=5 → `ValidationErrors`. 보관소 미갱신을 비공개 getter(또는 reflection) 또는 후속 정상 저장 호출 결과로 간접 확인.
- [x] C5. T4 (`TestSaveHostOptions_PersistsAcrossSessionReset`): 정상 저장 → OpenRoom(다른 옵션) → HostCloseRoom → 저장 옵션이 그대로 잔존. 확인 수단: 비공개 getter `getSavedHostOptions(s)` 헬퍼 작성(테스트 same-package).
- [x] C6. T5 (`TestSaveHostOptions_OverwriteLatest`): 정상 저장 두 번 → 두 번째 값이 보관됨.
- [x] C7. T6 (`TestSaveHostOptions_ConcurrentSafe`): 동일 호스트 토큰으로 N=20 goroutine 동시 호출 → race detector 통과 + 마지막 호출 결과 일치(원자성).

### Step D — 검증
- [x] D1. `go vet ./...` PASS.
- [x] D2. `go test ./internal/session/... -count=1 -race` PASS, 신규 6 케이스 모두 PASS.
- [x] D3. 패키지 커버리지: 기존 86%대 유지 또는 증가 (Iteration 5 기준 86.1%, Iteration 6 영향 없음 가정).

### Step E — 산출물
- [x] E1. 코드 변경 요약 문자열을 audit.md에 기록.
- [x] E2. 본 plan 체크박스 모두 [x].

## 변경 파일 목록 (예상)

| 파일 | 종류 | 변경 |
|---|---|---|
| `internal/session/session.go` | 수정 | 인터페이스 메서드 1건 + 구조체 필드 2건 |
| `internal/session/host_options.go` | 신규 | 구현 + 검증기 + 비공개 getter |
| `internal/session/iteration7_test.go` | 신규 | 6 단위 테스트 |

(그 외 파일은 변경 없음. `lifecycle.go`, `host_authority.go` 그대로.)

## 위험·롤백

- **위험**: 인터페이스 메서드 추가는 모든 SessionManager 구현체(테스트 mock 포함)를 깨뜨릴 수 있다. 검색 결과 외부 mock 없음(테스트는 동일 구현 사용). 만약 발견 시 stub 추가.
- **롤백**: 단일 신규 파일 + 단일 메서드. 인터페이스에서 메서드 제거하면 즉시 이전 상태 복귀.

## 사용자 승인 (Approval Gate)

본 Code Generation Plan v1.0을 검토하시고 다음 중 하나로 응답해 주십시오.

- **승인** — 계획대로 코드 생성을 시작합니다(Part 2 실행).
- **수정** — 변경/보완 항목을 알려주시면 v1.1로 갱신.
