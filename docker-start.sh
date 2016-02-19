#!/bin/bash -eu
docker rm ims-daemon
docker run --cap-add=NET_ADMIN -it --name "ims-daemon" -v $HOME/.aws:/mnt/ims ims /bin/bash -c "ifconfig eth0:0 169.254.169.254 netmask 255.255.255.255 && ./ims start --config=/mnt/ims/ims.conf $*"
