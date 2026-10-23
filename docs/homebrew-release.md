# Homebrew 출시 준비

후보 버전은 `v0.1.0`이다. 이 문서는 태그 생성, 라이선스 승인, 공개 릴리스 또는
tap 게시 승인을 대신하지 않는다. 기존 Build workflow와 압축파일 형식을 유지한다.

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
