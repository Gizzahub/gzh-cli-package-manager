# 9. Deployment Architecture

> gzh-cli-package-manager 아키텍처 문서 · [인덱스](README.md) · [ARCHITECTURE.md](../../ARCHITECTURE.md)

### 9.1 Build Artifacts

```
bin/
├── pmctl-darwin-amd64          # macOS Intel
├── pmctl-darwin-arm64          # macOS Apple Silicon
├── pmctl-linux-amd64           # Linux x86_64
├── pmctl-linux-arm64           # Linux ARM64
└── pmctl-windows-amd64.exe     # Windows (WSL2)
```

### 9.2 Installation Methods

**1. Homebrew (macOS/Linux)**:
```bash
brew tap gizzahub/tap
brew install pmctl
```

**2. Go Install**:
```bash
go install github.com/gizzahub/gzh-cli-package-manager/cmd/pmctl@latest
```

**3. Pre-built Binaries**:
```bash
curl -sfL https://raw.githubusercontent.com/gizzahub/gzh-cli-package-manager/main/install.sh | sh
```

### 9.3 Configuration Files

```
~/.config/pmctl/
├── config.yml                   # User configuration
├── state/
│   ├── manager_cache.json      # Manager detection cache
│   └── last_update.json        # Last update metadata
└── logs/
    └── pmctl.log               # Application logs
```
