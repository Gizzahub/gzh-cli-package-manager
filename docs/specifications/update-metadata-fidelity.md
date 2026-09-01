# Update 결과 version·download metadata fidelity

Release owner는 2026-09-01에 TASK-122 Recommended Direction을 승인했다.
이 문서는 그 승인을 구현 계약으로 고정한다.

## MVP 범위

Pilot manager는 **npm**과 **pacman**이다. 두 adapter는 update 전에
`ListPackages`로 현재 버전과 사용 가능 버전을 관측할 수 있다.

| Field | Pilot 정확도 | Provenance |
|---|---|---|
| package name | update 결과가 이름을 줄 때만 | `UpdateResult.UpdatedPackages` |
| OldVersion | 관측된 현재 버전만 | pre-update `ListPackages` |
| NewVersion | 관측된 available 버전만 | pre-update `ListPackages` |
| UpdateType | 관측한 두 버전으로만 계산 | derived, 직접 관측 아님 |
| SizeBytes | 현재 unavailable | adapter가 download size를 보고하지 않음. 추정 금지 |

현재 npm·pacman 성공 경로와 dry-run은 `UpdatedPackages`를 비운다. 이 경우
package row를 만들지 않고 `PackageCorrelation=unsupported`로 기록한다.
빈 이름 목록을 성공한 package metadata로 간주하지 않는다.

## Additive JSON 계약

기존 `PackageUpdate` 필드 이름(`Name`, `OldVersion`, `NewVersion`,
`UpdateType`, `SizeBytes`)은 유지한다. 호출자가 미관측 값을 실제 값으로
오해하지 않도록 presence 필드를 추가한다.

- `OldVersionPresence` / `NewVersionPresence`: `observed` 또는 `unavailable`
- `UpdateTypePresence`: 두 버전이 모두 관측되면 `derived`, 아니면 `unavailable`
- `SizeBytesPresence`: 현재 모든 manager에서 `unavailable`
- `PackageCorrelation`: `joined` · `partial` · `unobserved` · `unsupported` · `out_of_pilot` · `not_applicable`
- `MetadataPilot`: npm·pacman만 `true`

미관측 값:

- version 문자열은 빈 문자열이다. `"unknown"`을 넣지 않는다.
- `UpdateType`은 빈 값이다. `minor`를 기본값으로 넣지 않는다.
- `SizeBytes`의 `0`은 `SizeBytesPresence=unavailable`일 때 실제 0바이트가 아니다.

## 소비자 마이그레이션

이전 JSON은 `OldVersion`/`NewVersion`=`"unknown"`, `UpdateType`=`"minor"`,
`SizeBytes`=`0`을 채워 보냈다. 이 값들은 관측값이 아니었다.

- `"unknown"` 또는 `UpdateType=="minor"`만 보고 버전/유형을 단정하지 않는다.
- `SizeBytes==0`만 보고 다운로드가 없다고 단정하지 않는다.
- presence 필드가 `observed`/`derived`일 때만 해당 값을 사실로 사용한다.
- 기존 필드 이름은 그대로이므로 모르는 키를 무시하는 소비자는 계속 파싱할 수 있다.

## Pilot 밖 manager

brew, apt, pip, cargo, asdf, winget, scoop, chocolatey 및 기타 manager는
같은 정확도를 자동 적용하지 않는다. 이름이 있어도 version·update type·size는
`unavailable`이고 `PackageCorrelation=out_of_pilot`이다.

남은 범위:

- update 결과가 package 이름을 파싱하도록 npm·pacman을 확장할지
- download size를 실제로 보고하는 adapter가 생길 때 size presence를 observed로 올릴지
- dry-run preview를 빈 `UpdatedPackages`가 아니라 snapshot의 outdated 목록으로 보여줄지

이 항목은 후속 판단이며 이번 MVP 완료 기준이 아니다.

## 완료 기준

- npm·pacman만 ListPackages 스냅샷과 이름을 조인한다.
- 관측하지 않은 version·size·update type을 위조하지 않는다.
- 성공·실패·부분 metadata·dry-run 계약을 테스트로 고정한다.
- 기존 text 출력은 package name/count 중심을 유지한다.
