#provision:
#  ansible-galaxy install andrewrothstein.go-dev
#  ansible-playbook --ssh-common-args='-o StrictHostKeyChecking=no' ansible/bootstrap.yaml -i ansible/inventory

# https://gist.github.com/arunoda/7790979
#install-sshpass:
#  curl -L https://raw.githubusercontent.com/kadwanev/bigboybrew/master/Library/Formula/sshpass.rb > sshpass.rb && brew install sshpass.rb && rm sshpass.rb


COMMIT?=main

# resolve branch name to a commit
resolve-commit:
ifeq ("main", "main")
	$(eval COMMIT :=$(shell git ls-remote https://github.com/grafana/grafana HEAD -c7 | cut -f1))
	$(eval export COMMIT)
endif

tests: 
	git clone https://github.com/grafana/grafana-api-tests tests

build:
	git clone https://github.com/grafana/grafana-build build

update:
	cd tests && git pull
	cd build && git pull

build-commit: resolve-commit
ifeq (,$(wildcard artifacts/grafana-server-$(COMMIT)))
	cd build && go run ./cmd --verbose --grafana-ref=$(COMMIT) backend build --distro=linux/amd64
	#cd build && go run ./cmd --verbose --grafana-ref=$(COMMIT) build backend --distro=linux/amd64
	mv build/bin/linux/amd64/grafana-server artifacts/grafana-server-$(COMMIT)
endif

run:
