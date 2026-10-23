# Homebrew 출시 준비

최초 공개 버전은 `v0.1.0`이다. 이 문서는 이후 태그 생성, 라이선스 승인, 공개
릴리스 또는 tap 게시 승인을 대신하지 않는다. 기존 Build workflow와 압축파일
형식을 유지한다.

`make release-quality`는 기존 fmt·lint·race test에 PRODUCT.md의 전체 커버리지
85% 하한을 추가한다. 하한 미달은 출시 차단이며 테스트 제외로 우회하지 않는다.
Build workflow도 아카이브를 만들기 전에 같은 하한을 검사한다.

`make release-targets-check release-runtime-deps-check release-payload-check release-snapshot`은 5개 대상(macOS/Linux amd64·arm64, Windows amd64)의 바이너리와
라이선스 payload, 체크섬을 검증·생성한다. snapshot은 개발 버전이므로 공개 버전
메타데이터의 증거가 아니다. 배포 바이너리는 Go나 다른 공유 라이브러리를 요구하지
않으며, 업데이트할 패키지 매니저는 사용자가 별도로 설치해야 한다.

`make release-homebrew RELEASE_VERSION=0.1.0 RELEASE_CHECKSUMS=경로/checksums.txt`는
체크섬에서 macOS 두 아키텍처용 `build/homebrew/gz-pm.rb`를 만든다. 이 cask는
승인된 공개 `v0.1.0` 릴리스의 tar.gz를 받는다. 출시 전 snapshot 체크섬으로 만든
cask는 구조 확인용이며 공개 URL의 다운로드·설치 검증을 통과한 것이 아니다.
게시할 때는 해당 공개 릴리스에서 받은 체크섬을 사용해야 한다. 압축파일에는 실행
파일과 Apache-2.0 LICENSE, 제삼자 고지·라이선스가 함께 들어간다.

Build workflow는 브랜치에 `workflow_dispatch`와 `version=0.1.0`을 지정하면
게시 없이 압축파일·체크섬·cask 후보를 artifact로 올린다. tag ref에 dispatch하면
거부한다. Release job은 `v*` 태그 push에서만 draft를 만들며 cask를 tap에 쓰지
않는다. 최초 태그는 Go module proxy에 영구 기록될 수 있으므로 별도 승인 대상이다.

cask는 macOS 전용이다. Linux·Windows는 기존 압축파일 배포 범위를 유지한다.
공개 후 macOS Intel·Apple Silicon에서 다운로드, 체크섬, 설치, 버전과 제거를
확인한 다음 승인된 tap에 cask를 전달한다.

## 공개 산출물 게시 순서

태그 push로 실행되는 Build workflow가 공개 archive의 유일한 게시 경로다.
5개 플랫폼 빌드와 체크섬 생성, draft Release의 자산 업로드가 모두 끝난 뒤
내용을 검토하고 해당 draft를 공개한다. 같은 태그 workflow가 실행 중일 때
로컬 archive로 별도 Release를 만들거나 공개하면 두 게시 경로가 충돌한다.

업로드 직전 guard는 이미 공개된 Release를 거부한다. 조회 실패나 잘못된 응답도
실패로 처리한다. 공개된 자산을 교체해야 한다면 새 패치 버전 태그를 사용한다.
guard의 조회와 업로드는 원자적이지 않으므로 작업자는 workflow가 끝나기 전에
draft를 공개하지 않는다. tap의 checksum은 최종 공개 Release의 자산에서만 가져온다.

`v0.1.0`의 최종 공개 자산은 source `33f91f964826900e469f8e0f4179f61aed49a888`의
[Build 39958304066](https://github.com/Gizzahub/gzh-cli-package-manager/actions/runs/39958304066)에서
생성됐다. 앞서 만든 로컬 후보 archive는 최종 공개 자산이 아니다.
