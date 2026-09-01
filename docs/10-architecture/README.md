# 아키텍처 (Architecture)

gzh-cli-package-manager의 상세 아키텍처 문서입니다. 개요와 진입점은 루트
[`ARCHITECTURE.md`](../../ARCHITECTURE.md)를 참고하세요.

## Quick Navigation

| 질문 | 이동처 |
| ---- | ------ |
| 어떤 아키텍처 원칙을 따르는지 알고 싶다 | [10-principles.md](10-principles.md) |
| 레이어 구조(도메인/응용/인프라/표현)가 궁금하다 | [20-layers/](20-layers/README.md) |
| 디렉토리 배치 규칙을 알고 싶다 | [30-directory-structure.md](30-directory-structure.md) |
| 각 컴포넌트의 역할이 궁금하다 | [40-component-design.md](40-component-design.md) |
| 데이터 흐름을 추적하고 싶다 | [50-data-flow.md](50-data-flow.md) |
| 레이어 간 의존 규칙을 확인하고 싶다 | [60-dependency-rules.md](60-dependency-rules.md) |
| 테스트 전략과 커버리지 목표가 궁금하다 | [70-testing-strategy.md](70-testing-strategy.md) |
| 어떤 기술 스택을 쓰는지 알고 싶다 | [80-technology-stack.md](80-technology-stack.md) |
| 빌드·배포 방식을 알고 싶다 | [90-deployment.md](90-deployment.md) |
| 왜 이렇게 결정했는지 이력을 보고 싶다 | [adr/](adr/) |

## 목차

| 문서 | 원본 섹션 | 내용 |
| ---- | --------- | ---- |
| [10-principles.md](10-principles.md) | §1 | Architecture Principles — Clean Architecture, 의존성 역전 |
| [20-layers/](20-layers/README.md) | §2 | Layer Architecture — 4개 레이어 상세 (하위 폴더) |
| [30-directory-structure.md](30-directory-structure.md) | §3 | Directory Structure — 디렉토리 배치 |
| [40-component-design.md](40-component-design.md) | §4 | Component Design — 컴포넌트별 책임 |
| [50-data-flow.md](50-data-flow.md) | §5 | Data Flow — 요청 처리 흐름 |
| [60-dependency-rules.md](60-dependency-rules.md) | §6 | Dependency Rules — 레이어 간 의존 규칙 |
| [70-testing-strategy.md](70-testing-strategy.md) | §7 | Testing Strategy — 테스트 계층, 커버리지 |
| [80-technology-stack.md](80-technology-stack.md) | §8 | Technology Stack — 언어·라이브러리 선택 |
| [90-deployment.md](90-deployment.md) | §9 | Deployment Architecture — 빌드·배포 |
| [adr/](adr/) | — | Architecture Decision Records (9건; ADR-010 registry source-of-truth design 포함) |

> §2 Layer Architecture는 단일 파일 기준 14KB로 분할 기준(500줄/10KB)을 초과하여
> 레이어별로 [`20-layers/`](20-layers/README.md) 하위에 나누었습니다.

## 관련 문서

- 루트 [`ARCHITECTURE.md`](../../ARCHITECTURE.md) — 개요, Executive Summary
- [`PRD.md`](../../PRD.md) — 제품 요구사항
- [`REQUIREMENTS.md`](../../REQUIREMENTS.md) — 기능/비기능 요구사항
- [`docs/specifications/`](../specifications/) — 상세 스펙
