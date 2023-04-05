provision:
	ansible-playbook ansible/bootstrap.yaml -i ansible/inventory


# https://gist.github.com/arunoda/7790979
install-sshpass:
	curl -L https://raw.githubusercontent.com/kadwanev/bigboybrew/master/Library/Formula/sshpass.rb > sshpass.rb && brew install sshpass.rb && rm sshpass.rb
