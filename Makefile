#provision:
#  ansible-galaxy install andrewrothstein.go-dev
#  ansible-playbook --ssh-common-args='-o StrictHostKeyChecking=no' ansible/bootstrap.yaml -i ansible/inventory

# https://gist.github.com/arunoda/7790979
#install-sshpass:
#  curl -L https://raw.githubusercontent.com/kadwanev/bigboybrew/master/Library/Formula/sshpass.rb > sshpass.rb && brew install sshpass.rb && rm sshpass.rb

# default commit main
COMMIT?=main 

# get os and architecture lowercase
SYS_OS=$(shell uname -s | tr '[:upper:]' '[:lower:]')
SYS_ARCH=$(shell uname -m | tr '[:upper:]' '[:lower:]')

#default to system os/platform e.g. darwin/arm64
ARCH?=$(SYS_OS)/$(SYS_ARCH) 

# resolve main to latest commit
resolve-commit:
	@echo grafana-get-ref: $(COMMIT)
	echo ryan
	if [ "$(COMMIT)" = "main" ]; then \
		$(eval COMMIT := $(shell git ls-remote https://github.com/grafana/grafana HEAD -c7 | cut -f1)) \
		$(eval export COMMIT) \
		echo main resolved to: $(COMMIT) \
	else \
		echo "Strings are not equal";\
	fi	

resolve-arch: resolve-commit
	@echo arch: $(ARCH)
	$(eval ARTIFACT := grafana-server-$(COMMIT)-$(subst /,-,$(ARCH)))
	@echo $(ARTIFACT)
	$(eval ARTIFACT_PATH := ./artifacts/$(ARTIFACT))
	@echo artifact-path: $(ARTIFACT_PATH)

tests: 
	git clone https://github.com/grafana/grafana-api-tests tests

build:
	git clone https://github.com/grafana/grafana-build build

update:
	cd tests && git pull
	cd build && git pull

build-commit: tests build resolve-arch
	if [ -x "$(ARTIFACT_PATH)" ]; then \
		echo artifact exists, skipping build: $(ARTIFACT_PATH); \
	else \
		cd build && go run ./cmd --verbose --grafana-ref=$(COMMIT) backend build --distro=$(ARCH); \
		cd .. && mv build/bin/$(ARCH)/grafana-server $(ARTIFACT_PATH); \
	fi

boot-grafana: tests build resolve-arch
	mkdir -p workdir/
	rm -rf workdir/*
	cp -r grafana-conf workdir/conf
	# TODO cache this in the future
	curl -s -o https://raw.githubusercontent.com/grafana/grafana/$(COMMIT)/conf/defaults.ini workdir/conf/defaults.ini
	cp $(ARTIFACT_PATH) workdir/grafana
	./workdir/grafana

test-commit: tests build resolve-arch
	[ -x "$(ARTIFACT_PATH)" ] || make build-commit COMMIT=$(COMMIT) ARCH=$(ARCH)
	## START HERE - figuring out how to execute grafana
	$(ARTIFACT_PATH)
	cd tests && k6 run tests/dashboards.js

