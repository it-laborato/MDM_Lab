.PHONY: build clean clean-assets e2e-reset-db e2e-serve e2e-setup changelog db-reset db-backup db-restore check-go-cloner update-go-cloner help

export GO111MODULE=on
#  find . -name "*.go" -exec sed -i '' 's|github.com:it-laborato/MDM_Lab|github.com/it-laborato/MDM_Lab|g' {} + 
PATH := $(shell npm bin):$(PATH)
VERSION = $(shell git describe --tags --always --dirty)
BRANCH = $(shell git rev-parse --abbrev-ref HEAD)
REVISION = $(shell git rev-parse HEAD)
REVSHORT = $(shell git rev-parse --short HEAD)
USER = $(shell whoami)
DOCKER_IMAGE_NAME = mdmlabdm/mdmlab

ifdef GO_BUILD_RACE_ENABLED
GO_BUILD_RACE_ENABLED_VAR := true
else
GO_BUILD_RACE_ENABLED_VAR := false
endif

ifneq ($(OS), Windows_NT)
	# If on macOS, set the shell to bash explicitly
	ifeq ($(shell uname), Darwin)
		SHELL := /bin/bash
	endif

	# The output binary name is different on Windows, so we're explicit here
	OUTPUT = mdmlab

	# To populate version metadata, we use unix tools to get certain data
	GOVERSION = $(shell go version | awk '{print $$3}')
	NOW	= $(shell date +"%Y-%m-%d")
else
	# The output binary name is different on Windows, so we're explicit here
	OUTPUT = mdmlab.exe

	# To populate version metadata, we use windows tools to get the certain data
	GOVERSION_CMD = "(go version).Split()[2]"
	GOVERSION = $(shell powershell $(GOVERSION_CMD))
	NOW	= $(shell powershell Get-Date -format "yyy-MM-dd")
endif

ifndef CIRCLE_PR_NUMBER
	DOCKER_IMAGE_TAG = ${REVSHORT}
else
	DOCKER_IMAGE_TAG = dev-${CIRCLE_PR_NUMBER}-${REVSHORT}
endif

ifdef CIRCLE_TAG
	DOCKER_IMAGE_TAG = ${CIRCLE_TAG}
endif

LDFLAGS_VERSION = "\
	-X gitlab.it-laborato.ru/laborato/mdm-lab/server/version.appName=${APP_NAME} \
	-X gitlab.it-laborato.ru/laborato/mdm-lab/server/version.version=${VERSION} \
	-X gitlab.it-laborato.ru/laborato/mdm-lab/server/version.branch=${BRANCH} \
	-X gitlab.it-laborato.ru/laborato/mdm-lab/server/version.revision=${REVISION} \
	-X gitlab.it-laborato.ru/laborato/mdm-lab/server/version.buildDate=${NOW} \
	-X gitlab.it-laborato.ru/laborato/mdm-lab/server/version.buildUser=${USER} \
	-X gitlab.it-laborato.ru/laborato/mdm-lab/server/version.goVersion=${GOVERSION}"

all: build

define HELP_TEXT

  Makefile commands

	make deps         - Install dependent programs and libraries
	make generate     - Generate and bundle required all code
	make generate-go  - Generate and bundle required go code
	make generate-js  - Generate and bundle required js code
	make generate-dev - Generate and bundle required code in a watch loop

	make migration - create a database migration file (supply name=TheNameOfYourMigration)

	make generate-doc     - Generate updated API documentation for activities, osquery flags
	make dump-test-schema - update schema.sql from current migrations
	make generate-mock    - update mock data store

	make clean        - Clean all build artifacts
	make clean-assets - Clean assets only

	make build        - Build the code
	make package 	  - Build rpm and deb packages for linux

	make run-go-tests   - Run Go tests in specific packages
	make debug-go-tests - Debug Go tests in specific packages (with Delve)
	make test-js        - Run the JavaScript tests

	make lint         - Run all linters
	make lint-go      - Run the Go linters
	make lint-js      - Run the JavaScript linters
	make lint-scss    - Run the SCSS linters
	make lint-ts      - Run the TypeScript linters

	For use in CI:

	make test         - Run the full test suite (lint, Go and Javascript)
	make test-go      - Run the Go tests (all packages and tests)

endef

help:
	$(info $(HELP_TEXT))

.prefix:
	mkdir -p build/linux
	mkdir -p build/darwin

.pre-build:
	$(eval GOGC = off)
	$(eval CGO_ENABLED = 0)

.pre-mdmlab:
	$(eval APP_NAME = mdmlab)

.pre-mdmlabctl:
	$(eval APP_NAME = mdmlabctl)

build: mdmlab mdmlabctl

mdmlab: .prefix .pre-build .pre-mdmlab
	CGO_ENABLED=1 go build -race=${GO_BUILD_RACE_ENABLED_VAR} -tags full,fts5,netgo -o build/${OUTPUT} -ldflags ${LDFLAGS_VERSION} ./cmd/mdmlab

mdmlab-dev: GO_BUILD_RACE_ENABLED_VAR=true
mdmlab-dev: mdmlab

mdmlabctl: .prefix .pre-build .pre-mdmlabctl
	# Race requires cgo
	$(eval CGO_ENABLED := $(shell [[ "${GO_BUILD_RACE_ENABLED_VAR}" = "true" ]] && echo 1 || echo 0))
	$(eval mdmlabCTL_LDFLAGS := $(shell echo "${LDFLAGS_VERSION} ${EXTRA_mdmlabCTL_LDFLAGS}"))
	CGO_ENABLED=${CGO_ENABLED} go build -race=${GO_BUILD_RACE_ENABLED_VAR} -o build/mdmlabctl -ldflags="${mdmlabCTL_LDFLAGS}" ./cmd/mdmlabctl

mdmlabctl-dev: GO_BUILD_RACE_ENABLED_VAR=true
mdmlabctl-dev: mdmlabctl

lint-js:
	yarn lint

lint-go:
	golangci-lint run --exclude-dirs ./node_modules --timeout 15m

lint: lint-go lint-js

dump-test-schema:
	go run ./tools/dbutils ./server/datastore/mysql/schema.sql


# This is the base command to run Go tests.
# Wrap this to run tests with presets (see `run-go-tests` and `test-go` targets).
# PKG_TO_TEST: Go packages to test, e.g. "server/datastore/mysql".  Separate multiple packages with spaces.
# TESTS_TO_RUN: Name specific tests to run in the specified packages.  Leave blank to run all tests in the specified packages.
# GO_TEST_EXTRA_FLAGS: Used to specify other arguments to `go test`.
# GO_TEST_MAKE_FLAGS: Internal var used by other targets to add arguments to `go test`.
#
PKG_TO_TEST := "" # default to empty string; can be overridden on command line.
go_test_pkg_to_test := $(addprefix ./,$(PKG_TO_TEST)) # set paths for packages to test
dlv_test_pkg_to_test := $(addprefix gitlab.it-laborato.ru/laborato/mdm-lab/,$(PKG_TO_TEST)) # set URIs for packages to debug

DEFAULT_PKG_TO_TEST := ./cmd/... ./ee/... ./orbit/pkg/... ./orbit/cmd/orbit ./pkg/... ./server/... ./tools/...
ifeq ($(CI_TEST_PKG), main)
	CI_PKG_TO_TEST=$(shell go list ${DEFAULT_PKG_TO_TEST} | grep -v "server/datastore/mysql" | grep -v "cmd/mdmlabctl" | grep -v "server/vulnerabilities" | sed -e 's|gitlab.it-laborato.ru/laborato/mdm-lab/||g')
else ifeq ($(CI_TEST_PKG), integration)
	CI_PKG_TO_TEST="server/service"
else ifeq ($(CI_TEST_PKG), mysql)
	CI_PKG_TO_TEST="server/datastore/mysql/..."
else ifeq ($(CI_TEST_PKG), mdmlabctl)
	CI_PKG_TO_TEST="cmd/mdmlabctl/..."
else ifeq ($(CI_TEST_PKG), vuln)
	CI_PKG_TO_TEST="server/vulnerabilities/..."
else
	CI_PKG_TO_TEST=$(DEFAULT_PKG_TO_TEST)
endif

ci-pkg-list:
	@echo $(CI_PKG_TO_TEST)

.run-go-tests:
ifeq ($(PKG_TO_TEST), "")
		@echo "Please specify one or more packages to test with argument PKG_TO_TEST=\"/path/to/pkg/1 /path/to/pkg/2\"...";
else
		@echo Running Go tests with command:
		go test -tags full,fts5,netgo -run=${TESTS_TO_RUN} ${GO_TEST_MAKE_FLAGS} ${GO_TEST_EXTRA_FLAGS} -parallel 8 -coverprofile=coverage.txt -covermode=atomic -coverpkg=gitlab.it-laborato.ru/laborato/mdm-lab/... $(go_test_pkg_to_test)
endif

# This is the base command to debug Go tests.
# Wrap this to run tests with presets (see `debug-go-tests`)
# PKG_TO_TEST: Go packages to test, e.g. "server/datastore/mysql".  Separate multiple packages with spaces.
# TESTS_TO_RUN: Name specific tests to debug in the specified packages.  Leave blank to debug all tests in the specified packages.
# DEBUG_TEST_EXTRA_FLAGS: Internal var used by other targets to add arguments to `dlv test`.
# GO_TEST_EXTRA_FLAGS: Used to specify other arguments to `go test`.
.debug-go-tests:
ifeq ($(PKG_TO_TEST), "")
		@echo "Please specify one or more packages to debug with argument PKG_TO_TEST=\"/path/to/pkg/1 /path/to/pkg/2\"...";
else
		@echo Debugging tests with command:
		dlv test ${dlv_test_pkg_to_test} --api-version=2 --listen=127.0.0.1:61179 ${DEBUG_TEST_EXTRA_FLAGS} -- -test.v -test.run=${TESTS_TO_RUN} ${GO_TEST_EXTRA_FLAGS}
endif

# Command to run specific tests in development.  Can run all tests for one or more packages, or specific tests within packages.
run-go-tests:
	@MYSQL_TEST=1 REDIS_TEST=1 MINIO_STORAGE_TEST=1 SAML_IDP_TEST=1 NETWORK_TEST=1 make .run-go-tests GO_TEST_MAKE_FLAGS="-v"

debug-go-tests:
	@MYSQL_TEST=1 REDIS_TEST=1 MINIO_STORAGE_TEST=1 SAML_IDP_TEST=1 NETWORK_TEST=1 make .debug-go-tests

# Command used in CI to run all tests.
test-go: dump-test-schema generate-mock
	make .run-go-tests PKG_TO_TEST="$(CI_PKG_TO_TEST)"

analyze-go:
	go test -tags full,fts5,netgo -race -cover ./...

test-js:
	yarn test

test: lint test-go test-js

generate: clean-assets generate-js generate-go

generate-ci:
	NODE_OPTIONS=--openssl-legacy-provider NODE_ENV=development yarn run webpack
	make generate-go

generate-js: clean-assets .prefix
	NODE_ENV=production yarn run webpack --progress

generate-go: .prefix
	go run github.com/kevinburke/go-bindata/go-bindata@v3 -pkg=bindata -tags full \
		-o=server/bindata/generated.go \
		frontend/templates/ assets/... server/mail/templates

# we first generate the webpack bundle so that bindata knows to atch the
# output bundle file. then, generate debug bindata source file. finally, we
# run webpack in watch mode to continuously re-generate the bundle
generate-dev: .prefix
	NODE_ENV=development yarn run webpack --progress
	go run github.com/kevinburke/go-bindata/go-bindata@v3 -debug -pkg=bindata -tags full \
		-o=server/bindata/generated.go \
		frontend/templates/ assets/... server/mail/templates
	NODE_ENV=development yarn run webpack --progress --watch

generate-mock: .prefix
	go generate gitlab.it-laborato.ru/laborato/mdm-lab/server/mock gitlab.it-laborato.ru/laborato/mdm-lab/server/mock/mockresult gitlab.it-laborato.ru/laborato/mdm-lab/server/service/mock

generate-doc: .prefix
	go generate gitlab.it-laborato.ru/laborato/mdm-lab/server/mdmlab
	go generate gitlab.it-laborato.ru/laborato/mdm-lab/server/service/osquery_utils

deps: deps-js deps-go

deps-js:
	yarn

deps-go:
	go mod download

# check that the generated files in tools/cloner-check/generated_files match
# the current version of the cloneable structures.
check-go-cloner:
	go run ./tools/cloner-check/main.go --check

# update the files in tools/cloner-check/generated_files with the current
# version of the cloneable structures.
update-go-cloner:
	go run ./tools/cloner-check/main.go --update

migration:
	go run ./server/goose/cmd/goose -dir server/datastore/mysql/migrations/tables create $(name)
	gofmt -w server/datastore/mysql/migrations/tables/*_$(name)*.go

clean: clean-assets
	rm -rf build vendor
	rm -f assets/bundle.js

clean-assets:
	git clean -fx assets

docker-build-release: xp-mdmlab xp-mdmlabctl
	docker build -t "${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG}" .
	docker tag "${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG}" mdmlabdm/mdmlab:${VERSION}
	docker tag "${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG}" mdmlabdm/mdmlab:latest

docker-push-release: docker-build-release
	docker push "${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG}"
	docker push mdmlabdm/mdmlab:${VERSION}
	docker push mdmlabdm/mdmlab:latest

mdmlabctl-docker: xp-mdmlabctl
	docker build -t mdmlabdm/mdmlabctl --platform=linux/amd64 -f tools/mdmlabctl-docker/Dockerfile .

bomutils-docker:
	cd tools/bomutils-docker && docker build -t mdmlabdm/bomutils --platform=linux/amd64 -f Dockerfile .

wix-docker:
	cd tools/wix-docker && docker build -t mdmlabdm/wix --platform=linux/amd64 -f Dockerfile .

.pre-binary-bundle:
	rm -rf build/binary-bundle
	mkdir -p build/binary-bundle/linux
	mkdir -p build/binary-bundle/darwin

xp-mdmlab: .pre-binary-bundle .pre-mdmlab generate
	CGO_ENABLED=1 GOOS=linux go build -tags full,fts5,netgo -trimpath -o build/binary-bundle/linux/mdmlab -ldflags ${LDFLAGS_VERSION} ./cmd/mdmlab
	CGO_ENABLED=1 GOOS=darwin go build -tags full,fts5,netgo -trimpath -o build/binary-bundle/darwin/mdmlab -ldflags ${LDFLAGS_VERSION} ./cmd/mdmlab
	CGO_ENABLED=1 GOOS=windows go build -tags full,fts5,netgo -trimpath -o build/binary-bundle/windows/mdmlab.exe -ldflags ${LDFLAGS_VERSION} ./cmd/mdmlab

xp-mdmlabctl: .pre-binary-bundle .pre-mdmlabctl generate-go
	CGO_ENABLED=0 GOOS=linux go build -trimpath -o build/binary-bundle/linux/mdmlabctl -ldflags ${LDFLAGS_VERSION} ./cmd/mdmlabctl
	CGO_ENABLED=0 GOOS=darwin go build -trimpath -o build/binary-bundle/darwin/mdmlabctl -ldflags ${LDFLAGS_VERSION} ./cmd/mdmlabctl
	CGO_ENABLED=0 GOOS=windows go build -trimpath -o build/binary-bundle/windows/mdmlabctl.exe -ldflags ${LDFLAGS_VERSION} ./cmd/mdmlabctl

binary-bundle: xp-mdmlab xp-mdmlabctl
	cd build/binary-bundle && zip -r mdmlab.zip darwin/ linux/ windows/
	cd build/binary-bundle && mkdir mdmlabctl-macos && cp darwin/mdmlabctl mdmlabctl-macos && tar -czf mdmlabctl-macos.tar.gz mdmlabctl-macos
	cd build/binary-bundle && mkdir mdmlabctl-linux && cp linux/mdmlabctl mdmlabctl-linux && tar -czf mdmlabctl-linux.tar.gz mdmlabctl-linux
	cd build/binary-bundle && mkdir mdmlabctl-windows && cp windows/mdmlabctl.exe mdmlabctl-windows && tar -czf mdmlabctl-windows.tar.gz mdmlabctl-windows
	cd build/binary-bundle && cp windows/mdmlabctl.exe . && zip mdmlabctl.exe.zip mdmlabctl.exe
	cd build/binary-bundle && shasum -a 256 mdmlab.zip mdmlabctl.exe.zip mdmlabctl-macos.tar.gz mdmlabctl-windows.tar.gz mdmlabctl-linux.tar.gz

# Build orbit/mdmlabd mdmlabd_tables extension
mdmlabd-tables-windows:
	GOOS=windows GOARCH=amd64 go build -o mdmlabd_tables_windows.exe ./orbit/cmd/mdmlabd_tables
mdmlabd-tables-linux:
	GOOS=linux GOARCH=amd64 go build -o mdmlabd_tables_linux.ext ./orbit/cmd/mdmlabd_tables
mdmlabd-tables-linux-arm64:
	GOOS=linux GOARCH=arm64 go build -o mdmlabd_tables_linux_arm64.ext ./orbit/cmd/mdmlabd_tables
mdmlabd-tables-darwin:
	GOOS=darwin GOARCH=amd64 go build -o mdmlabd_tables_darwin.ext ./orbit/cmd/mdmlabd_tables
mdmlabd-tables-darwin_arm64:
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=1 go build -o mdmlabd_tables_darwin_arm64.ext ./orbit/cmd/mdmlabd_tables
mdmlabd-tables-darwin-universal: mdmlabd-tables-darwin mdmlabd-tables-darwin_arm64
	lipo -create mdmlabd_tables_darwin.ext mdmlabd_tables_darwin_arm64.ext -output mdmlabd_tables_darwin_universal.ext
mdmlabd-tables-all: mdmlabd-tables-windows mdmlabd-tables-linux mdmlabd-tables-darwin-universal mdmlabd-tables-linux-arm64
mdmlabd-tables-clean:
	rm -f mdmlabd_tables_windows.exe mdmlabd_tables_linux.ext mdmlabd_tables_linux_arm64.ext mdmlabd_tables_darwin.ext mdmlabd_tables_darwin_arm64.ext mdmlabd_tables_darwin_universal.ext

.pre-binary-arch:
ifndef GOOS
	@echo "GOOS is Empty. Try use to see valid GOOS/GOARCH platform: go tool dist list. Ex.: make binary-arch GOOS=linux GOARCH=arm64"
	@exit 1;
endif
ifndef GOARCH
	@echo "GOARCH is Empty. Try use to see valid GOOS/GOARCH platform: go tool dist list. Ex.: make binary-arch GOOS=linux GOARCH=arm64"
	@exit 1;
endif


binary-arch: .pre-binary-arch .pre-binary-bundle .pre-mdmlab
	mkdir -p build/binary-bundle/${GOARCH}-${GOOS}
	CGO_ENABLED=1 GOARCH=${GOARCH} GOOS=${GOOS} go build -tags full,fts5,netgo -o build/binary-bundle/${GOARCH}-${GOOS}/mdmlab -ldflags ${LDFLAGS_VERSION} ./cmd/mdmlab
	CGO_ENABLED=0 GOARCH=${GOARCH} GOOS=${GOOS} go build -tags full,fts5,netgo -o build/binary-bundle/${GOARCH}-${GOOS}/mdmlabctl -ldflags ${LDFLAGS_VERSION} ./cmd/mdmlabctl
	cd build/binary-bundle/${GOARCH}-${GOOS} && tar -czf mdmlabctl-${GOARCH}-${GOOS}.tar.gz mdmlabctl mdmlab


# Drop, create, and migrate the e2e test database
e2e-reset-db:
	docker compose exec -T mysql_test bash -c 'echo "drop database if exists e2e; create database e2e;" | MYSQL_PWD=toor mysql -uroot'
	./build/mdmlab prepare db --mysql_address=localhost:3307  --mysql_username=root --mysql_password=toor --mysql_database=e2e

e2e-setup:
	./build/mdmlabctl config set --context e2e --address https://localhost:8642 --tls-skip-verify true
	./build/mdmlabctl setup --context e2e --email=admin@example.com --password=password123# --org-name='mdmlab Test' --name Admin
	./build/mdmlabctl user create --context e2e --email=maintainer@example.com --name maintainer --password=password123# --global-role=maintainer
	./build/mdmlabctl user create --context e2e --email=observer@example.com --name observer --password=password123# --global-role=observer
	./build/mdmlabctl user create --context e2e --email=sso_user@example.com --name "SSO user" --sso=true

# Setup e2e test environment and pre-populate database with software and vulnerabilities fixtures.
#
# Use in lieu of `e2e-setup` for tests that depend on these fixtures
e2e-setup-with-software:
	curl 'https://localhost:8642/api/v1/setup' \
		--data-raw '{"server_url":"https://localhost:8642","org_info":{"org_name":"mdmlab Test"},"admin":{"admin":true,"email":"admin@example.com","name":"Admin","password":"password123#","password_confirmation":"password123#"}}' \
		--compressed \
		--insecure
	./tools/backup_db/restore_e2e_software_test.sh

e2e-serve-free: e2e-reset-db
	./build/mdmlab serve --mysql_address=localhost:3307 --mysql_username=root --mysql_password=toor --mysql_database=e2e --server_address=0.0.0.0:8642

e2e-serve-premium: e2e-reset-db
	./build/mdmlab serve  --dev_license --mysql_address=localhost:3307 --mysql_username=root --mysql_password=toor --mysql_database=e2e --server_address=0.0.0.0:8642

# Associate a host with a mdmlab Desktop token.
#
# Usage:
# make e2e-set-desktop-token host_id=1 token=foo
e2e-set-desktop-token:
	docker compose exec -T mysql_test bash -c 'echo "INSERT INTO e2e.host_device_auth (host_id, token) VALUES ($(host_id), \"$(token)\") ON DUPLICATE KEY UPDATE token=VALUES(token)" | MYSQL_PWD=toor mysql -uroot'

changelog:
	sh -c "find changes -type f | grep -v .keep | xargs -I {} sh -c 'grep \"\S\" {}; echo' > new-CHANGELOG.md"
	sh -c "cat new-CHANGELOG.md CHANGELOG.md > tmp-CHANGELOG.md && rm new-CHANGELOG.md && mv tmp-CHANGELOG.md CHANGELOG.md"
	sh -c "git rm changes/*"

changelog-orbit:
	$(eval TODAY_DATE := $(shell date "+%b %d, %Y"))
	@echo -e "## Orbit $(version) ($(TODAY_DATE))\n" > new-CHANGELOG.md
	sh -c "find orbit/changes -type file | grep -v .keep | xargs -I {} sh -c 'grep \"\S\" {} | sed -E "s/^-/*/"; echo' >> new-CHANGELOG.md"
	sh -c "cat new-CHANGELOG.md orbit/CHANGELOG.md > tmp-CHANGELOG.md && rm new-CHANGELOG.md && mv tmp-CHANGELOG.md orbit/CHANGELOG.md"
	sh -c "git rm orbit/changes/*"

changelog-chrome:
	$(eval TODAY_DATE := $(shell date "+%b %d, %Y"))
	@echo -e "## mdmlabd-chrome $(version) ($(TODAY_DATE))\n" > new-CHANGELOG.md
	sh -c "find ee/mdmlabd-chrome/changes -type file | grep -v .keep | xargs -I {} sh -c 'grep \"\S\" {}; echo' >> new-CHANGELOG.md"
	sh -c "cat new-CHANGELOG.md ee/mdmlabd-chrome/CHANGELOG.md > tmp-CHANGELOG.md && rm new-CHANGELOG.md && mv tmp-CHANGELOG.md ee/mdmlabd-chrome/CHANGELOG.md"
	sh -c "git rm ee/mdmlabd-chrome/changes/*"

# Updates the documentation for the currently released versions of mdmlabd components in mdmlab's TUF.
mdmlabd-tuf:
	sh -c 'echo "<!-- DO NOT EDIT. This document is automatically generated by running \`make mdmlabd-tuf\`. -->\n# tuf.mdmlabctl.com\n\nFollowing are the currently deployed versions of mdmlabd components on the \`stable\` and \`edge\` channel.\n" > orbit/TUF.md'
	sh -c 'echo "## \`stable\`\n" >> orbit/TUF.md'
	sh -c 'go run tools/tuf/status/tuf-status.go channel-version -channel stable -format markdown >> orbit/TUF.md'
	sh -c 'echo "\n## \`edge\`\n" >> orbit/TUF.md'
	sh -c 'go run tools/tuf/status/tuf-status.go channel-version -channel edge -format markdown >> orbit/TUF.md'

###
# Development DB commands
###

# Reset the development DB
db-reset:
	docker compose exec -T mysql bash -c 'echo "drop database if exists mdmlab; create database mdmlab;" | MYSQL_PWD=toor mysql -uroot'
	./build/mdmlab prepare db --dev

# Back up the development DB to file
db-backup:
	./tools/backup_db/backup.sh

# Restore the development DB from file
db-restore:
	./tools/backup_db/restore.sh

# Generate osqueryd.app.tar.gz bundle from osquery.io.
#
# Usage:
# make osqueryd-app-tar-gz version=5.1.0 out-path=.
osqueryd-app-tar-gz:
ifneq ($(shell uname), Darwin)
	@echo "Makefile target osqueryd-app-tar-gz is only supported on macOS"
	@exit 1
endif
	$(eval TMP_DIR := $(shell mktemp -d))
	curl -L https://github.com/osquery/osquery/releases/download/$(version)/osquery-$(version).pkg --output $(TMP_DIR)/osquery-$(version).pkg
	pkgutil --expand $(TMP_DIR)/osquery-$(version).pkg $(TMP_DIR)/osquery_pkg_expanded
	rm -rf $(TMP_DIR)/osquery_pkg_payload_expanded
	mkdir -p $(TMP_DIR)/osquery_pkg_payload_expanded
	tar xf $(TMP_DIR)/osquery_pkg_expanded/Payload --directory $(TMP_DIR)/osquery_pkg_payload_expanded
	$(TMP_DIR)/osquery_pkg_payload_expanded/opt/osquery/lib/osquery.app/Contents/MacOS/osqueryd --version
	tar czf $(out-path)/osqueryd.app.tar.gz -C $(TMP_DIR)/osquery_pkg_payload_expanded/opt/osquery/lib osquery.app
	rm -r $(TMP_DIR)

# Generate nudge.app.tar.gz bundle from nudge repo.
#
# Usage:
# make nudge-app-tar-gz version=1.1.10.81462 out-path=.
nudge-app-tar-gz:
ifneq ($(shell uname), Darwin)
	@echo "Makefile target nudge-app-tar-gz is only supported on macOS"
	@exit 1
endif
	$(eval TMP_DIR := $(shell mktemp -d))
	curl -L https://github.com/macadmins/nudge/releases/download/v$(version)/Nudge-$(version).pkg --output $(TMP_DIR)/nudge-$(version).pkg
	pkgutil --expand $(TMP_DIR)/nudge-$(version).pkg $(TMP_DIR)/nudge_pkg_expanded
	mkdir -p $(TMP_DIR)/nudge_pkg_payload_expanded
	tar xvf $(TMP_DIR)/nudge_pkg_expanded/nudge-$(version).pkg/Payload --directory $(TMP_DIR)/nudge_pkg_payload_expanded
	$(TMP_DIR)/nudge_pkg_payload_expanded/Nudge.app/Contents/MacOS/Nudge --version
	tar czf $(out-path)/nudge.app.tar.gz -C $(TMP_DIR)/nudge_pkg_payload_expanded/ Nudge.app
	rm -r $(TMP_DIR)

# Generate swiftDialog.app.tar.gz bundle from the swiftDialog repo.
#
# Usage:
# make swift-dialog-app-tar-gz version=2.2.1 build=4591 out-path=.
swift-dialog-app-tar-gz:
ifneq ($(shell uname), Darwin)
	@echo "Makefile target swift-dialog-app-tar-gz is only supported on macOS"
	@exit 1
endif
	# locking the version of swiftDialog to 2.2.1-4591 as newer versions
	# might have layout issues.
ifneq ($(version), 2.2.1)
	@echo "Version is locked at 2.1.0, see comments in Makefile target for details"
	@exit 1
endif

ifneq ($(build), 4591)
	@echo "Build version is locked at 4591, see comments in Makefile target for details"
	@exit 1
endif
	$(eval TMP_DIR := $(shell mktemp -d))
	curl -L https://github.com/swiftDialog/swiftDialog/releases/download/v$(version)/dialog-$(version)-$(build).pkg --output $(TMP_DIR)/swiftDialog-$(version).pkg
	pkgutil --expand $(TMP_DIR)/swiftDialog-$(version).pkg $(TMP_DIR)/swiftDialog_pkg_expanded
	mkdir -p $(TMP_DIR)/swiftDialog_pkg_payload_expanded
	tar xvf $(TMP_DIR)/swiftDialog_pkg_expanded/tmp-package.pkg/Payload --directory $(TMP_DIR)/swiftDialog_pkg_payload_expanded
	$(TMP_DIR)/swiftDialog_pkg_payload_expanded/Library/Application\ Support/Dialog/Dialog.app/Contents/MacOS/Dialog --version
	tar czf $(out-path)/swiftDialog.app.tar.gz -C $(TMP_DIR)/swiftDialog_pkg_payload_expanded/Library/Application\ Support/Dialog/ Dialog.app
	rm -rf $(TMP_DIR)

# Generate escrowBuddy.pkg bundle from the Escrow Buddy repo.
#
# Usage:
# make escrow-buddy-pkg version=1.0.0 out-path=.
escrow-buddy-pkg:
	curl -L https://github.com/macadmins/escrow-buddy/releases/download/v$(version)/Escrow.Buddy-$(version).pkg --output $(out-path)/escrowBuddy.pkg


# Build and generate desktop.app.tar.gz bundle.
#
# Usage:
# mdmlab_DESKTOP_APPLE_AUTHORITY=foo mdmlab_DESKTOP_VERSION=0.0.1 make desktop-app-tar-gz
#
# Output: desktop.app.tar.gz
desktop-app-tar-gz:
ifneq ($(shell uname), Darwin)
	@echo "Makefile target desktop-app-tar-gz is only supported on macOS"
	@exit 1
endif
	go run ./tools/desktop macos

mdmlab_DESKTOP_VERSION ?= unknown

# Build desktop executable for Windows.
# This generates desktop executable for Windows that includes versioninfo binary properties
# These properties can be displayed when right-click on the binary in Windows Explorer.
# See: https://docs.microsoft.com/en-us/windows/win32/menurc/versioninfo-resource
# To sign this binary with a certificate, use signtool.exe or osslsigncode tool
#
# Usage:
# mdmlab_DESKTOP_VERSION=0.0.1 make desktop-windows
#
# Output: mdmlab-desktop.exe
desktop-windows:
	go run ./orbit/tools/build/build-windows.go -version $(mdmlab_DESKTOP_VERSION) -input ./orbit/cmd/desktop -output mdmlab-desktop.exe

# Build desktop executable for Linux.
#
# Usage:
# mdmlab_DESKTOP_VERSION=0.0.1 make desktop-linux
#
# Output: desktop.tar.gz
desktop-linux:
	docker build -f Dockerfile-desktop-linux -t desktop-linux-builder .
	docker run --rm -v $(shell pwd):/output desktop-linux-builder /bin/bash -c "\
		mkdir -p /output/mdmlab-desktop && \
		go build -o /output/mdmlab-desktop/mdmlab-desktop -ldflags "-X=main.version=$(mdmlab_DESKTOP_VERSION)" /usr/src/mdmlab/orbit/cmd/desktop && \
		cd /output && \
		tar czf desktop.tar.gz mdmlab-desktop && \
		rm -r mdmlab-desktop"

# Build desktop executable for Linux ARM.
#
# Usage:
# mdmlab_DESKTOP_VERSION=0.0.1 make desktop-linux-arm64
#
# Output: desktop.tar.gz
desktop-linux-arm64:
	docker build -f Dockerfile-desktop-linux -t desktop-linux-builder .
	docker run --rm -v $(shell pwd):/output desktop-linux-builder /bin/bash -c "\
		mkdir -p /output/mdmlab-desktop && \
		GOARCH=arm64 go build -o /output/mdmlab-desktop/mdmlab-desktop -ldflags "-X=main.version=$(mdmlab_DESKTOP_VERSION)" /usr/src/mdmlab/orbit/cmd/desktop && \
		cd /output && \
		tar czf desktop.tar.gz mdmlab-desktop && \
		rm -r mdmlab-desktop"

# Build orbit executable for Windows.
# This generates orbit executable for Windows that includes versioninfo binary properties
# These properties can be displayed when right-click on the binary in Windows Explorer.
# See: https://docs.microsoft.com/en-us/windows/win32/menurc/versioninfo-resource
# To sign this binary with a certificate, use signtool.exe or osslsigncode tool
#
# Usage:
# ORBIT_VERSION=0.0.1 make orbit-windows
#
# Output: orbit.exe
orbit-windows:
	go run ./orbit/tools/build/build-windows.go -version $(ORBIT_VERSION) -input ./orbit/cmd/orbit -output orbit.exe

# db-replica-setup setups one main and one read replica MySQL instance for dev/testing.
#	- Assumes the docker containers are already running (tools/mysql-replica-testing/docker-compose.yml)
# 	- MySQL instance listening on 3308 is the main instance.
# 	- MySQL instance listening on 3309 is the read instance.
#	- Sets a delay of 1s for replication.
db-replica-setup:
	$(eval MYSQL_REPLICATION_USER := replicator)
	$(eval MYSQL_REPLICATION_PASSWORD := rotacilper)
	MYSQL_PWD=toor mysql --host 127.0.0.1 --port 3309 -uroot -AN -e "stop slave; reset slave all;"
	MYSQL_PWD=toor mysql --host 127.0.0.1 --port 3308 -uroot -AN -e "drop user if exists '$(MYSQL_REPLICATION_USER)'; create user '$(MYSQL_REPLICATION_USER)'@'%' identified by '$(MYSQL_REPLICATION_PASSWORD)'; grant replication slave on *.* to '$(MYSQL_REPLICATION_USER)'@'%'; flush privileges;"
	$(eval MAIN_POSITION := $(shell MYSQL_PWD=toor mysql --host 127.0.0.1 --port 3308 -uroot -e 'show master status \G' | grep Position | grep -o '[0-9]*'))
	$(eval MAIN_FILE := $(shell MYSQL_PWD=toor mysql --host 127.0.0.1 --port 3308 -uroot -e 'show master status \G' | grep File | sed -n -e 's/^.*: //p'))
	MYSQL_PWD=toor mysql --host 127.0.0.1 --port 3309 -uroot -AN -e "change master to master_port=3306,master_host='mysql_main',master_user='$(MYSQL_REPLICATION_USER)',master_password='$(MYSQL_REPLICATION_PASSWORD)',master_log_file='$(MAIN_FILE)',master_log_pos=$(MAIN_POSITION);"
	if [ "${mdmlab_MYSQL_IMAGE}" == "mysql:8.0" ]; then MYSQL_PWD=toor mysql --host 127.0.0.1 --port 3309 -uroot -AN -e "change master to get_master_public_key=1;"; fi
	MYSQL_PWD=toor mysql --host 127.0.0.1 --port 3309 -uroot -AN -e "change master to master_delay=1;"
	MYSQL_PWD=toor mysql --host 127.0.0.1 --port 3309 -uroot -AN -e "start slave;"

# db-replica-reset resets the main MySQL instance.
db-replica-reset: mdmlab
	MYSQL_PWD=toor mysql --host 127.0.0.1 --port 3308 -uroot -e "drop database if exists mdmlab; create database mdmlab;"
	mdmlab_MYSQL_ADDRESS=127.0.0.1:3308 ./build/mdmlab prepare db --dev

# db-replica-run runs mdmlab serve with one main and one read MySQL instance.
db-replica-run: mdmlab
	mdmlab_MYSQL_ADDRESS=127.0.0.1:3308 mdmlab_MYSQL_READ_REPLICA_ADDRESS=127.0.0.1:3309 mdmlab_MYSQL_READ_REPLICA_USERNAME=mdmlab mdmlab_MYSQL_READ_REPLICA_DATABASE=mdmlab mdmlab_MYSQL_READ_REPLICA_PASSWORD=insecure ./build/mdmlab serve --dev --dev_license


rename-file:
	grep -rilZ --null 'mdmlab' . | xargs -0 perl -i -pe 's/\b(mdmlab)\b/mdmlab/gi'
rename:
	find . -depth -iname "*mdmlab*" -execdir bash -c 'old_name="$1" new_name=$(echo "$old_name" | sed "s/[fF][lL][eE][eE][tT]/mdmlab/g")  mv -- "$old_name" "$new_name"' _ {} \;
