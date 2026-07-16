# Architecture: gzh-cli-package-manager

> **Version**: 1.1.0
> **Last Updated**: 2026-07-16
> **Status**: Draft

> 상세 설계는 주제별로 [`docs/10-architecture/`](docs/10-architecture/README.md)에
> 분할되어 있습니다. 이 문서는 개요와 진입점 역할을 합니다.

---

## Executive Summary

`gzh-cli-package-manager` follows **Clean Architecture** principles with **Hexagonal (Ports & Adapters)** pattern to ensure long-term maintainability, testability, and extensibility.

**Key Architectural Decisions**:
1. **Domain-Driven Design**: Business logic isolated from infrastructure
2. **Dependency Inversion**: High-level modules don't depend on low-level details
3. **Single Responsibility**: Each layer has one reason to change
4. **Testability**: Every layer independently testable with mocks

**Architecture Guarantees**:
- 5+ year lifespan with manageable complexity
- Easy addition of new package managers (plugin architecture)
- Swappable infrastructure (CLI → GUI, YAML → Database)
- 90%+ test coverage achievable

---

## Document Map

| 질문 | 이동처 |
| ---- | ------ |
| 어떤 아키텍처 원칙을 따르는지 알고 싶다 | [10-principles.md](docs/10-architecture/10-principles.md) |
| 레이어 구조(도메인/응용/인프라/표현)가 궁금하다 | [20-layers/](docs/10-architecture/20-layers/README.md) |
| 디렉토리 배치 규칙을 알고 싶다 | [30-directory-structure.md](docs/10-architecture/30-directory-structure.md) |
| 각 컴포넌트의 역할이 궁금하다 | [40-component-design.md](docs/10-architecture/40-component-design.md) |
| 데이터 흐름을 추적하고 싶다 | [50-data-flow.md](docs/10-architecture/50-data-flow.md) |
| 레이어 간 의존 규칙을 확인하고 싶다 | [60-dependency-rules.md](docs/10-architecture/60-dependency-rules.md) |
| 테스트 전략과 커버리지 목표가 궁금하다 | [70-testing-strategy.md](docs/10-architecture/70-testing-strategy.md) |
| 어떤 기술 스택을 쓰는지 알고 싶다 | [80-technology-stack.md](docs/10-architecture/80-technology-stack.md) |
| 빌드·배포 방식을 알고 싶다 | [90-deployment.md](docs/10-architecture/90-deployment.md) |
| 왜 이렇게 결정했는지 이력을 보고 싶다 | [adr/](docs/10-architecture/adr/) |

전체 목차: [`docs/10-architecture/README.md`](docs/10-architecture/README.md)

---

## References

- **PRD**: `/PRD.md`
- **Requirements**: `/REQUIREMENTS.md`
- **Architecture details**: `/docs/10-architecture/`
- **ADRs**: `/docs/10-architecture/adr/`
- **Specifications**: `/docs/specifications/`

---

**Document Control**:
- **Author**: Claude Code (AI-assisted)
- **Architect**: TBD
- **Reviewers**: TBD
- **Approval**: TBD
- **Next Review**: 2025-02-05
