# 9. Deployment Architecture

> gzh-cli-package-manager 아키텍처 문서 · [인덱스](README.md) · [ARCHITECTURE.md](../../ARCHITECTURE.md)

### 9.1 Build Artifacts

```
bin/
├── gz-pm-darwin-amd64          # macOS Intel
├── gz-pm-darwin-arm64          # macOS Apple Silicon
├── gz-pm-linux-amd64           # Linux x86_64
├── gz-pm-linux-arm64           # Linux ARM64
└── gz-pm-windows-amd64.exe     # Windows (WSL2)
```

### 9.2 Installation Methods

**1. Homebrew (macOS/Linux)**:

첫 승인 릴리스 이후 제공할 예정이며 현재 formula와 stable tag는 없다.

```bash
brew tap gizzahub/tap
brew install gz-pm
```

**2. Go Install**:
```bash
go install github.com/gizzahub/gzh-cli-package-manager/cmd/gz-pm@latest
```

**3. Pre-built Binaries**:

첫 승인 태그 이후 GitHub Releases의 `gz-pm-<os>-<arch>` artifact를 사용한다.

### 9.3 Configuration Files

```
~/.config/gz-pm/
├── config.yml                   # User configuration
├── state/
│   ├── manager_cache.json      # Manager detection cache
│   └── last_update.json        # Last update metadata
└── logs/
    └── gz-pm.log               # Application logs
```
