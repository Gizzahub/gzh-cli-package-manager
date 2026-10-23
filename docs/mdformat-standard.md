# Markdown formatter 적용 기록

## 현재 계약

선택기는 저장소 전용 Python `scripts/mdformat-files.py`이며, installer는 healthy
runtime도 Python >=3.11인지 확인한다. `UV_PYTHON`이 unset일 때만 >=3.11 runtime을
기본 선택하고 명시값은 덮어쓰지 않는다. changed 검사는 `origin/master` merge-base,
committed·staged·unstaged·untracked Markdown과 type-change를 포함하며 index/worktree
불일치는 실패한다. Git index 120000 symlink(Windows에서 일반 파일로 materialize된 경우
포함)는 제외한다.

CI는 PR base SHA, dev branch의 `origin/master`, trunk push의 before SHA를 사용하고
수동 실행은 전체 검사다. Ubuntu와 Windows native smoke는 각각 10분 초기 예산을 쓴다.
constraints와 top-level version/plugin 검사는 installer 계약으로 계속 유지한다.

## 적용 계약

ISSUE-352의 채택 묶음은 mdformat 1.0.0, mdformat-gfm 1.0.0,
mdformat-frontmatter 2.1.2다. `make install-mdformat`은 `uv tool install`로
저장소의 `bin/tools`에 설치하고, 정확한 플러그인 집합 및
`.make/mdformat-constraints.txt`의 다섯 하위 의존성을 검증한다.
버전 명령이 정상 문자열을 출력하고 실패해도 거부한다.

기존 `make fmt-md`를 저장소 전용 실행 파일로 전환했다.
Git이 추적하거나 ignore하지 않은 일반 Markdown만 선택하며,
symlink·디렉터리와 ignored 도구 venv는 제외한다.
공백·개행은 NUL 구분으로 보존하고, `./` 접두사로 `-` 시작 파일명도 처리한다.
Git 조회·formatter 실패는 Make 실패로 전파한다. Bash와 NUL 입력을 지원하는
xargs가 필요하다. 전역 formatter와 기존 Go 도구 pin은 변경하지 않는다.

## 범위와 기존 상태

`fmt-md`는 전체 Markdown을 명시적으로 정렬한다. `quality`는 기존 Go fmt·lint·test를
유지하면서 `origin/master` 이후 Markdown을 읽기 전용으로 검사한다. 전체 검사는
`make format-md-all-check`로 명시적으로만 실행한다.

일반 Markdown 45개의 이전 묶음·새 묶음 출력은 모두 byte 동일했다.
그중 42개는 기존부터 canonical 서식과 달랐으며 이번에 일괄 정렬하지 않았다.
Git 목록의 나머지 두 Markdown 경로는 symlink로 선택에서 제외했다.
정렬 전 비교는 `tmp/issue352/regression.json`에 기록했다.

구현·독립 리뷰 요구 등급은 strong이다. 사용자 지시로 외부 CLI를 분업에 사용했다.
`claudz --model sonnet`의 로그는 glm-5.3-flash 매핑을 보여 주며,
워커는 Make 설정 세 파일의 수정만 담당했다. 부모가 실제 diff를 확인하고
파일명 회귀를 보강했다. Grok의 초기 읽기 전용 조사는 10턴 제한으로 종료돼
최종 검증으로 계산하지 않았다. 외부 실행 로그는 `tmp/issue352`에 보존한다.
범위를 줄여 코드 조각을 제공한 Grok 교차 분석은 정상 종료했다.
분석 제안은 실제 Make 진단으로 확인하며, 모델의 추정을 곧바로 결함으로 기록하지 않는다.

## 검증 범위

좁은 회귀는 YAML frontmatter·표·멱등성, 버전 순서 및 잘못된 버전·누락·추가
플러그인·exit 42 거부, 하위 의존성 불일치, 공백·개행·`-` 경로,
symlink·디렉터리·ignored venv 제외, Git 및 formatter 실패 전파를 검사한다.
임시 진단 코드는 저장소의 ignored `tmp/issue352`에 두었다.

커밋 검사는 기존 AGENTS.md의 `make lint-diff` 후 `make quality`를 따른다.
quality는 fmt·lint·전체 race test를 포함하며 테스트 timeout은 기존 5분이다.
실행별 외부 진단 제한은 임시 1800초이며 프로젝트 timeout·병렬도는 변경하지 않았다.
상세 메트릭 수집 설계는 이번 범위에 포함하지 않는다.

base `2a7f8c548d596da5cc70184b87c45e311b06747b`의 현재 입력에서
`make lint-diff`는 exit 0(3.834초), `make quality`는 exit 0(34.458초)이었다.
quality 실행 뒤 Go 파일 변경은 없었다. `tmp/issue352/checks.json` 및 각 실행 로그에
증거를 보존하며, 이 수치를 성능 개선이나 별도 release 검증으로 해석하지 않는다.

Windows 실제 실행, Python 3.11+ 선택 강제, 제품 설치·release 검증은 하지 않았다.
source 전달과 보드 전달은 실제 native 결과를 얻은 뒤 별도로 기록해야 한다.
