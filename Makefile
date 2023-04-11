#provision:
#  ansible-galaxy install andrewrothstein.go-dev
#  ansible-playbook --ssh-common-args='-o StrictHostKeyChecking=no' ansible/bootstrap.yaml -i ansible/inventory

# https://gist.github.com/arunoda/7790979
#install-sshpass:
#  curl -L https://raw.githubusercontent.com/kadwanev/bigboybrew/master/Library/Formula/sshpass.rb > sshpass.rb && brew install sshpass.rb && rm sshpass.rb


COMMIT?=main
ARCH?=linux/amd64

# resolve branch name to a commit
resolve-commit:
	@echo grafana-get-ref: $(COMMIT)
ifeq ("$(COMMIT)", "main")
	$(eval COMMIT :=$(shell git ls-remote https://github.com/grafana/grafana HEAD -c7 | cut -f1))
	$(eval export COMMIT)
	@echo main resolved to: $(COMMIT)
endif

resolve-arch: resolve-commit
	@echo arch: $(ARCH)
	$(eval ARTIFACT := grafana-server-$(COMMIT)-$(subst /,-,$(ARCH)))
	@echo $(ARTIFACT)
	$(eval ARTIFACT_PATH := artifacts/$(ARTIFACT))

tests: 
	git clone https://github.com/grafana/grafana-api-tests tests

build:
	git clone https://github.com/grafana/grafana-build build

update:
	cd tests && git pull
	cd build && git pull

build-commit: resolve-arch
ifeq (,$(wildcard $(ARTIFACT_PATH)))
	cd build && go run ./cmd --verbose --grafana-ref=$(COMMIT) backend build --distro=$(ARCH)
	mv build/bin/linux/amd64/grafana-server $(ARTIFACT_PATH)
endif

test-commit: resolve-arch
ifeq (,$(wildcard $(ARTIFACT_PATH)))
	@MAKE build-commit COMMIT=$(COMMIT) ARCH=$(ARCH)
endif
