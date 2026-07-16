# 레이어 아키텍처 (Layer Architecture)

원본 `ARCHITECTURE.md` §2에 해당합니다. Clean Architecture의 4개 레이어를
레이어별 문서로 나누어 두었습니다. 상위 인덱스는
[`../README.md`](../README.md)를 참고하세요.

의존 방향은 항상 바깥에서 안쪽(Presentation → Infrastructure → Application →
Domain)이며, Domain은 어떤 레이어에도 의존하지 않습니다. 규칙의 상세는
[`../60-dependency-rules.md`](../60-dependency-rules.md)를 보세요.

## 목차

| 레이어 | 원본 섹션 | 위치 | 문서 |
| ------ | --------- | ---- | ---- |
| Domain | §2.1 | `pkg/domain` | [10-domain.md](10-domain.md) |
| Application | §2.2 | `pkg/application` | [20-application.md](20-application.md) |
| Infrastructure | §2.3 | `pkg/infrastructure` | [30-infrastructure.md](30-infrastructure.md) |
| Presentation | §2.4 | `cmd/pmctl` | [40-presentation.md](40-presentation.md) |

## 관련 문서

- [`../10-principles.md`](../10-principles.md) — 레이어 분리의 근거가 되는 원칙
- [`../60-dependency-rules.md`](../60-dependency-rules.md) — 레이어 간 의존 규칙
- [`../adr/002-clean-architecture.md`](../adr/002-clean-architecture.md) — Clean Architecture 채택 결정
- [`../adr/003-hexagonal-ports-adapters.md`](../adr/003-hexagonal-ports-adapters.md) — Ports & Adapters 결정
