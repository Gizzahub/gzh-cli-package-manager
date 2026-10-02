# mise 도구 갱신 정책

`gz-pm update --managers mise`는 이미 설치된 mise에 도구 갱신을 위임합니다.
mise 자체를 설치하거나 갱신하지 않습니다. 설정 변경으로 필요한 도구를 설치하는
`mise install`은 환경 설치 절차에서 별도로 실행합니다.

## 정책과 범위 변경

| `--strategy` | mise 도구의 갱신 범위 |
| --- | --- |
| `stable` (기본) | mise의 기본 릴리스 해석으로 선언된 요청 안에서 갱신 |
| `latest` | 최신 SemVer 후보 (`--bump`와 함께 사용하면 시험판 포함) |
| `minor` | 현재 설치 버전의 메이저를 유지하고 마이너·패치 갱신 |
| `micro` | 현재 설치 버전의 메이저·마이너를 유지하고 패치 갱신 |
| `fixed` | 갱신을 실행하지 않음 |

모든 정책은 기본적으로 mise 설정의 버전 요청을 유지합니다. `latest`를 선택해도
자동으로 요청 범위를 넓히지 않습니다. `--bump`를 명시하면 요청 범위를 넘어선
갱신과 mise 설정 변경을 허용합니다. `minor`와 `micro`의 현재 버전 제한은
`--bump`를 지정해도 유지됩니다. `fixed`와 `--bump`는 함께 사용할 수 없습니다.
시험판은 숫자 접두 요청의 기본 mise 해석에서 활성화되지 않을 수 있으므로,
`latest`에서도 `--bump`를 지정한 경우에만 선택하고 정확한 시험판 요청을 저장합니다.

예를 들어 현재 `1.8.1`이 설치되어 있고 설정 요청이 `"1"`인 도구는 `micro`로
1.8 계열의 최신 패치만 설치합니다. 요청이 `"1.8.1"`이면 bump 없이는 유지하고,
`micro --bump`로는 1.8 계열의 최신 패치를 설치하고 설정을 그 버전으로 바꿉니다.
정책별로 버전을 지정하는 갱신은 정확한 선택 버전을 mise에 전달하므로 bump 시
기존 접두 요청도 정확한 버전으로 바뀝니다. `stable --bump`는 mise의 기본 bump
동작에 위임하여 기존 요청의 정밀도를 유지합니다.

`latest/minor/micro`는 현재 설치된 활성 도구에 적용하며 숫자 SemVer 버전과
숫자 요청(`"22"`, `"22.14"`, `"22.14.0"`), `latest` 요청을 지원합니다.
`lts`, git ref, 비-SemVer 버전, 여러 활성 버전을 사용하는 도구는 자동 해석하지
않고 갱신 전에 오류를 냅니다. 필요한 경우 `stable`로 mise의 고유 해석에 위임합니다.
혼합 설정에서는 `--mise-tools go,pnpm,jq`로 지원되는 도구만 선택할 수 있습니다.
선택한 도구가 활성·설치 상태가 아니거나 이름이 비어 있거나 중복되면 오류를 냅니다.
정책 갱신은 전체 도구의 대상 버전을 계획한 뒤 실행하며 다운그레이드하지 않습니다.
네이티브 명령 실행 중 실패하면 이미 갱신된 도구를 되돌리지는 않습니다.

```sh
# 기본: 선언된 요청 유지
gz-pm update --managers mise --dry-run

# 현재 메이저·마이너 유지
gz-pm update --managers mise --strategy micro --dry-run

# 혼합 설정에서 특정 도구만 선택
gz-pm update --managers mise --mise-tools go,pnpm,jq --strategy micro --dry-run

# 패치 범위 안에서 정확한 핀 변경까지 허용
gz-pm update --managers mise --strategy micro --bump --dry-run

# 정식 릴리스로 범위 변경
gz-pm update --managers mise --strategy stable --bump --dry-run

# 시험판까지 고려한 범위 변경
gz-pm update --managers mise --strategy latest --bump --dry-run
```

`--dry-run`은 선택한 정책을 계획하고 네이티브 mise dry-run을 호출합니다.
도구 설치·요청 변경은 실행하지 않습니다. 실제 실행에서는 `--dry-run`을 제거합니다.
갱신된 이전 버전을 즉시 지우지 않도록 모든 갱신 명령에 `--no-prune`을 전달합니다.

## 설정 탐색과 저장된 정책

기본 mise 설정 범위는 현재 디렉터리에서 mise가 탐색하는 로컬·상위·전역 설정입니다.
`--mise-dir /path/to/project`로 탐색 기준 디렉터리를 지정하고,
`--mise-local`로 프로젝트 로컬 설정의 도구만 대상으로 제한할 수 있습니다.
전역 설정을 갱신할 때는 로컬 mise 설정이 없는 디렉터리를 기준으로 실행합니다.

gz-pm 정책 파일은 `$XDG_CONFIG_HOME/gz-pm/config.yaml`을 사용합니다.
환경 변수가 없으면 `~/.config/gz-pm/config.yaml`을 사용합니다.
`--config /path/to/config.yaml`로 다른 파일을 선택할 수 있습니다.
기본 파일은 없어도 되지만 명시한 파일이 없거나 잘못되면 오류를 냅니다.

```yaml
version: 1
defaults:
  strategy: stable
managers:
  mise:
    strategy: micro
    bump: false
    directory: /path/to/project
    local: true
    tools: [go, pnpm, jq]
  brew:
    strategy: fixed
```

우선순위는 명시한 CLI 옵션 → 매니저별 설정 → 기본 설정 → 내장 기본값입니다.
`--bump=false`와 `--mise-local=false`도 설정 파일의 true 값을 덮어씁니다.
설정 파일은 gz-pm이 자동으로 작성하거나 변경하지 않습니다.

`bump`, `directory`, `local`, `tools`는 `managers.mise`에서만 설정할 수 있습니다.
`tools`를 생략하면 모든 활성 도구를 대상으로 합니다.
`--bump`는 `--managers mise`와 함께 사용합니다. `--all`에서 mise의 요청 변경이
필요하면 설정 파일의 `managers.mise.bump: true`로 mise에만 허용합니다.

## 다른 매니저와의 관계

현재 다른 매니저는 `minor/micro`의 버전 제한을 구현하지 않았습니다. 해당 정책이
선택된 다른 매니저에 적용되면 전체 갱신을 시작하기 전에 오류를 냅니다.
다른 매니저의 `stable/latest`는 기존 네이티브 일괄 갱신 동작이며 두 정책의
릴리스 필터 차이를 보장하지 않습니다. 기존 `minor` 호출이 네이티브 무제한 갱신으로
실행되던 동작은 이제 오류로 바뀝니다.

mise 도구만 갱신하려면 `--managers mise`를 사용합니다. `--all`은 Homebrew,
전역 npm·cargo 등도 대상으로 하므로 환경 설치 후 mise 적용을 대신하는 명령으로
사용하지 않습니다.

참조: [mise upgrade](https://mise.jdx.dev/cli/upgrade.html),
[mise ls](https://mise.jdx.dev/cli/ls.html),
[mise ls-remote](https://mise.jdx.dev/cli/ls-remote.html).
