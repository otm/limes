#!/bin/bash -eu
# Example Usage: docker-start.sh --mfa=123456
docker rm ims-daemon
docker run --net="host" --cap-add=NET_ADMIN -it --name "ims-daemon" -v $HOME/.aws:/mnt/ims ims /bin/bash -c "ifconfig lo:0 169.254.169.254 netmask 255.255.255.255 && ./ims start --config=/mnt/ims/ims.conf $*"
