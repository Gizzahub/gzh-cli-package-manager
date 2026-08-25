# 1. Architecture Principles

> gzh-cli-package-manager 아키텍처 문서 · [인덱스](README.md) · [ARCHITECTURE.md](../../ARCHITECTURE.md)

### 1.1 Clean Architecture Layers

```
┌─────────────────────────────────────────────┐
│  Presentation Layer (cmd/gz-pm)             │  ← Frameworks & Drivers
│  - CLI Commands (Cobra)                     │
│  - Formatters (Text, JSON)                  │
│  - Input Validation                         │
├─────────────────────────────────────────────┤
│  Application Layer (pkg/application)        │  ← Use Cases
│  - Use Case Orchestration                   │
│  - Business Workflows                       │
│  - Input/Output Ports                       │
├─────────────────────────────────────────────┤
│  Domain Layer (pkg/domain)                  │  ← Entities & Business Rules
│  - Core Entities (Manager, Package)         │
│  - Business Logic (Update Strategies)       │
│  - Repository Interfaces                    │
│  - Domain Services                          │
├─────────────────────────────────────────────┤
│  Infrastructure Layer (pkg/infrastructure)  │  ← External Interfaces
│  - Package Manager Adapters                 │
│  - Command Execution                        │
│  - File System Operations                   │
│  - Repository Implementations               │
└─────────────────────────────────────────────┘
```

**Dependency Direction**: Outer layers depend on inner layers, never the reverse.

---

### 1.2 Hexagonal Architecture (Ports & Adapters)

```
                        Application Core
                    ┌─────────────────────┐
                    │   Domain Layer      │
                    │  (Business Rules)   │
                    └──────────┬──────────┘
                               │
                    ┌──────────┴──────────┐
                    │  Application Layer  │
                    │   (Use Cases)       │
                    └──┬────────────────┬─┘
                       │                │
            ┌──────────┴───┐     ┌─────┴──────────┐
            │ Input Ports  │     │ Output Ports   │
            │ (Driven)     │     │ (Driving)      │
            └──────┬───────┘     └────────┬───────┘
                   │                      │
       ┌───────────┴─────────┐  ┌────────┴─────────────┐
       │  Primary Adapters   │  │ Secondary Adapters   │
       │  - CLI (Cobra)      │  │ - Homebrew Adapter   │
       │  - REST API (future)│  │ - ASDF Adapter       │
       └─────────────────────┘  │ - File System        │
                                │ - Command Executor   │
                                └──────────────────────┘
```

**Benefits**:
- Application core is framework-agnostic
- Easy to swap adapters (e.g., CLI → GUI)
- Testable without external dependencies

---

### 1.3 SOLID Principles Application

**Single Responsibility Principle (SRP)**:
- Each package has one reason to change
- Example: `pkg/domain/manager` only knows about package manager concepts

**Open/Closed Principle (OCP)**:
- New package managers added without modifying core
- Strategy pattern for update strategies

**Liskov Substitution Principle (LSP)**:
- All manager adapters implement same interface
- Swappable without breaking client code

**Interface Segregation Principle (ISP)**:
- Small, focused interfaces (ManagerDetector, VersionParser)
- Clients depend only on methods they use

**Dependency Inversion Principle (DIP)**:
- High-level (use cases) depend on abstractions (ports)
- Low-level (adapters) implement abstractions
