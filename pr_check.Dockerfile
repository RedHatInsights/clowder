from fedora:32
run dnf install -y openssh-clients git podman make which go jq
run ln -s /usr/bin/podman /usr/bin/docker
copy pr_check_inner.sh .
run chmod 775 pr_check_inner.sh
