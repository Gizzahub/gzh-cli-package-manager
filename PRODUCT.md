# Product Goals (No-PRD)

**Project**: gzh-cli-package-manager (`gz-pm` binary)
**Doc Type**: Goals + Constraints + Quality Gates
**Status**: Active
**Last Updated**: 2026-07-16

______________________________________________________________________

## Product Intent

gzh-cli-package-manager is a **multi-package-manager update orchestrator**. Through
one CLI (and importable command tree) it:

- detects which package managers are installed and reports their health,
- drives bulk updates through per-manager adapters instead of reimplementing them
  (SOUL 신념 1: 감싸되 대체하지 않는다),
- and keeps every mutation behind `--dry-run` and explicit opt-in guards.

This is a feature-library project — a single PRODUCT.md is sufficient. It
replaces a PRD.

| 제공하는 것 (Is)                              | 되지 않을 것 (Is Not)                       |
| --------------------------------------------- | ------------------------------------------- |
| 설치된 매니저 감지·상태 보고                  | 패키지 매니저 자체 구현                     |
| 여러 매니저의 일괄 업데이트 오케스트레이션    | 매니저 자체를 설치해 주는 부트스트래퍼      |
| dry-run·전략(latest/stable/minor/fixed) 선택  | 개별 패키지 install/remove/search           |
| text·json 출력, 매니저별 종료 코드            | GUI·TUI·REST API·클라우드 동기화            |

______________________________________________________________________

## Goals (Measurable Targets)

G1. **Detection breadth**

- Target: 10개 매니저 감지 (brew·asdf·npm·pip·cargo·apt·pacman·winget·scoop·choco)
  — 현재 10/10 등록 완료

G2. **Update parity (감지 = 실동작)**

- Target: 감지되는 매니저는 모두 `update`가 실제 동작해야 한다 (10/10)
- 현재 **4/10** — brew·scoop·winget·choco만 실구현. npm·pip·cargo·asdf는
  `"update not yet implemented"` 스텁, apt·pacman은 어댑터 미연결
  ("no adapter found"). **이 격차가 본 리포 최우선 과제다.**

G3. **Safe by default**

- Target: 모든 파괴적 경로는 exec 이전에 `--dry-run` 분기; sudo 사용 0건;
  shell 보간(`sh -c`) 0건 — 현재 3항목 모두 충족

G4. **Clean Architecture 경계**

- Target: domain 계층의 외부 import 0건 (stdlib만) — 현재 충족

G5. **Test reliability**

- Target: 커버리지 >= 85% (현재 81.9%; `cmd/pm`·`pkg/domain/cleanup`는 0%)

______________________________________________________________________

## Non-Goals (Explicitly Out of Scope)

- No 패키지 매니저 자체 구현 — 네이티브 매니저에 위임한다
- No 매니저 자체 설치 (`bootstrap`은 감지·dry-run 안내까지만)
- No 개별 패키지 단위 작업 (install/remove/search) — 일괄 업데이트가 정체성
- No 권한 상승 — sudo를 호출하지 않는다
- No GUI·TUI·REST API·Prometheus 메트릭·팀 공유 설정

______________________________________________________________________

## Guardrails and Technical Constraints

**Architecture**

- Clean Architecture: `domain`/`application`/`infrastructure` 계층 분리
- domain은 stdlib만 import한다 (외부 의존 0건)
- 외부 명령은 `exec.CommandContext(ctx, cmd, args...)`로만 실행 — `sh -c` 금지
  (command injection 방지)

**Dependency Boundaries**

- `gzh-cli-core`만 의존 가능; 다른 feature 라이브러리 의존 금지 (GUIDELINES §2)

**License (문서화된 예외)**

- 본 리포는 **Apache-2.0**을 유지한다. GUIDELINES §3은 패밀리 관행상 MIT를
  권장하지만, package-manager의 Apache-2.0은 **기존 예외로 명시 승인**되어 있다
  (GUIDELINES §3·§5 — 리포별 예외는 해당 PRODUCT.md Guardrails에 기록한다).
  변경 시 특허 조항 상실 검토가 선행되어야 한다.

**Compatibility**

- Go 1.25+ (`go.mod` go 1.25.7; devbox 툴체인 1.26); CGO 미사용 (순수 정적 빌드)

**Safety**

- `--dry-run`은 어떤 exec보다 먼저 분기한다
- conda 환경에서 pip 업데이트는 자동 차단 — `--pip-allow-conda` 명시 opt-in 필요
- 확인 프롬프트·롤백은 현재 없음 — 파괴적 범위를 넓히기 전에 도입해야 한다

______________________________________________________________________

## Quality Gates (Release Readiness)

**Build and Lint**

- `make quality` (fmt + lint + test) pass with no warnings

**Testing**

- `make test-coverage` pass; 커버리지 >= 85%

**Correctness**

- 감지되는 매니저의 `update`가 스텁을 반환하지 않는다 (G2 — 현재 미충족)

**Docs**

- README·CLAUDE.md의 명령·상태 서술이 실제 코드와 일치한다 (현재 미충족:
  "implementation pending" 서술이 stale)

______________________________________________________________________

## Decision Rules

- 새 매니저 지원은 **감지 + Update 실구현**을 함께 포함해야 한다 — 감지만 추가하는
  변경은 G2 격차를 키우므로 거절된다
- 매니저 재구현은 SOUL 게이트 1(재발명 금지)에서 거절된다 — 감싸되 대체하지 않는다
- 새 기능은 SOUL.md 4-게이트(틈 · 라이브러리 · 대량/전환 · 날카로움)를 통과해야 한다
- Guardrails 위반은 문서화된 예외를 요구한다 (본 문서의 Apache-2.0 항목이 그 예)
- Quality Gates 미충족 시 릴리스는 차단된다

______________________________________________________________________

**End of Document**
