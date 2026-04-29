# Build Instructions — mafia-game

**작성일**: 2026-04-26
**대상**: 모든 단위 (U1~U5) 통합 빌드 → 단일 Go 바이너리

본 문서는 mafia-game을 처음부터 빌드하는 절차를 단계별로 정의합니다. 호스트 PC 운영자가 실행하는 진입점이기도 하며, CI 자동화에도 동일한 절차가 적용됩니다.

---

## 0. 사전 요건 (Prerequisites)

| 도구 | 버전 | 비고 |
|---|---|---|
| Go | **1.25.0+** | U2가 modernc.org/sqlite 요구로 1.22 → 1.25 갱신 |
| Node.js | **20+** | Vite 5 요구 |
| npm | 10+ | Node에 동봉 |
| OS | macOS / Linux / Windows | gorilla/websocket + modernc.org/sqlite 모두 cross-platform |
| RAM / 디스크 | ≥ 2 GB free / ≥ 1 GB free | node_modules ~150 MB, Go 캐시 ~500 MB |
| 네트워크 | 빌드 시 인터넷 (의존 fetch) | 운영 시 LAN만 필요 |

**단위별 사전 요건**:
- U1, U2, U3, U4: Go만 (Pure Go drivers, cgo 불필요)
- U5: Node.js + npm

---

## 1. 저장소 클론 + 디렉터리 구조 확인

```bash
git clone https://github.com/saltware/mafia-game.git
cd mafia-game

# 핵심 디렉터리:
# cmd/mafia-game/      — Composition Root (main.go + embed dist)
# internal/game/       — U1 도메인
# internal/{session,announce,persistence}/ — U2
# internal/transport/ws/ — U3
# internal/transport/http/ — U4
# web/                 — U5 React SPA 소스
# aidlc-docs/          — 문서
```

**확인할 placeholder**: `cmd/mafia-game/web/dist/index.html` 존재해야 함 (commit됨, 항상 있음).

---

## 2. 단위별 빌드 (Bottom-up)

### 2.1 U1~U4 (Go 백엔드)

```bash
go mod download                    # 의존 캐시
go build ./...                     # 모든 패키지 컴파일 검증 (산출 없음)
go vet ./...                       # 정적 분석
gofmt -l ./internal/ ./cmd/        # 포맷 검증 — 출력 비어야 통과
```

**기대 결과**: 모든 명령이 0 종료 코드. 에러 발생 시 어느 단위 단계에서 발생했는지 audit 로그 확인.

### 2.2 U5 (React SPA)

```bash
cd web

npm install                        # 1회 (node_modules 설치, ~30초)
npm run typecheck                  # tsc --noEmit
npm run lint                       # eslint
npm test                           # vitest 32 테스트
npm run build                      # tsc + vite build → ../cmd/mafia-game/web/dist/

cd ..
```

**기대 결과**: `cmd/mafia-game/web/dist/`에 `index.html` + `assets/main-{hash}.js` + `assets/main-{hash}.css` 산출.

> ⚠️ npm 첫 실행 시 인터넷 필요. 이후 `node_modules`가 캐시되어 빠름.

### 2.3 통합 빌드 (단일 명령)

```bash
make all
```

또는 수동:

```bash
cd web && npm install && npm run build && cd ..
go build -o mafia-game ./cmd/mafia-game
```

---

## 3. 산출물 검증

### 3.1 단일 바이너리

```bash
file mafia-game
# 기대: Mach-O 64-bit executable arm64 (macOS)
#       또는 ELF 64-bit LSB executable (Linux)

ls -lh mafia-game
# 기대: ~16 MB
```

### 3.2 동봉된 SPA 자산 확인

```bash
go build -ldflags="-s -w" -o mafia-game ./cmd/mafia-game
# 또는 그냥 build 후
./mafia-game --port 8080 &
sleep 1
curl -s http://localhost:8080/healthz
# 기대: ok

curl -s http://localhost:8080/ | head -3
# 기대: <!doctype html><html lang="ko">...

kill %1
```

---

## 4. 정적 검증 게이트 (CI 권장)

| 단계 | 명령 | 통과 조건 |
|---|---|---|
| Go vet | `go vet ./...` | 0 issue |
| Go fmt | `gofmt -l ./internal/ ./cmd/` | 빈 출력 |
| TS strict | `cd web && npm run typecheck` | 0 error |
| ESLint | `cd web && npm run lint` | 0 error |
| Go build | `go build ./...` | 모든 패키지 컴파일 성공 |
| 단일 바이너리 | `go build ./cmd/mafia-game` | 산출 OK |

---

## 5. 운영 배포 (LAN 배포)

```bash
# 1) 빌드 (개발자 PC)
make all

# 2) 호스트 PC로 복사
scp mafia-game host-pc:/home/host/

# 3) 호스트 PC에서 실행
ssh host-pc
mkdir -p ~/data
./mafia-game --port 8080 --db ./data/mafia.db
```

**시작 시 콘솔 출력 예시**:
```
mafia-game listening on:
  http://192.168.1.42:8080
```

플레이어들은 `http://192.168.1.42:8080/play`로 접속.
호스트는 `http://localhost:8080/public`으로 진입 후 닉네임 등록 → 호스트 권한 자동 획득.

---

## 6. 트러블슈팅

| 증상 | 원인 | 해결 |
|---|---|---|
| `go: pattern all:web/dist: no matching files found` | placeholder 부재 | `cmd/mafia-game/web/dist/index.html` git에 포함되어야 함 |
| `npm install` 실패 | 인터넷 연결 / Node 버전 | Node ≥ 20 확인 |
| 단일 바이너리 실행 시 "frontend not built" 503 | placeholder만 동봉됨 | `cd web && npm run build` 후 다시 `go build` |
| WebSocket 연결 실패 (브라우저 콘솔) | 포트 / 방화벽 | 호스트 PC `:8080` open 확인 |
| LAN URL 출력 안 됨 (localhost만) | net.IsPrivate() 매칭 IP 없음 | VPN 끄기 / 유선 LAN 연결 |

---

## 7. 검증 체크리스트

- [x] Prerequisites 명시 (Go 1.25, Node 20)
- [x] 단위별 빌드 명령 (Go + npm)
- [x] 통합 빌드 단일 명령 (Vite → Go embed → 단일 바이너리)
- [x] 산출물 검증 (file 타입 / 크기 / /healthz)
- [x] 정적 검증 게이트 (vet/fmt/typecheck/lint)
- [x] 운영 배포 절차 (호스트 PC 복사 + LAN URL)
- [x] 트러블슈팅 표
