# 5. Data Flow

> gzh-cli-package-manager 아키텍처 문서 · [인덱스](README.md) · [ARCHITECTURE.md](../../ARCHITECTURE.md)

### 5.1 Update Command Flow

```
User Input
    ↓
CLI Command (Presentation)
    ↓
Input Validation
    ↓
UpdateAllManagersUseCase (Application)
    ↓
ManagerRepository.FindInstalled() → Adapters (Infrastructure)
    ↓
For each manager:
    UpdateStrategy.SelectVersion() (Domain)
    ↓
    Adapter.Update() → Shell Execution (Infrastructure)
    ↓
    Parse Output → Domain Entities
    ↓
BuildResponse (Application)
    ↓
Format Output (Presentation)
    ↓
Display to User
```

### 5.2 Sequence Diagram: Update All Managers

```mermaid
sequenceDiagram
    participant User
    participant CLI
    participant UseCase
    participant Repo
    participant Adapter
    participant Executor

    User->>CLI: pmctl update --all
    CLI->>CLI: Parse flags, validate
    CLI->>UseCase: Execute(UpdateAllRequest)

    UseCase->>Repo: FindInstalled()
    Repo->>Adapter: Detect() for each manager
    Adapter->>Executor: Execute("which brew")
    Executor-->>Adapter: Result
    Adapter-->>Repo: Manager{ID: "brew"}
    Repo-->>UseCase: []Manager

    loop For each manager
        UseCase->>Adapter: Update(strategy)
        Adapter->>Executor: Execute("brew update && brew upgrade")
        Executor-->>Adapter: Output
        Adapter->>Adapter: Parse output
        Adapter-->>UseCase: UpdateResult
    end

    UseCase->>UseCase: Build summary
    UseCase-->>CLI: UpdateAllResponse
    CLI->>CLI: Format (text/json)
    CLI-->>User: Display results
```
