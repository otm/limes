#!/bin/bash -eu
# Launches the limes daemon inside a docker container. Mounts ~/.aws as a conf dir, so put your limes.conf at ~/.aws/limes.conf.
# Example Usage: docker-start.sh --mfa=123456
docker rm limes-daemon || true
docker run --net="host" --cap-add=NET_ADMIN -it --name "limes-daemon" -v $HOME/.aws:/mnt/limes-conf limes /bin/bash -c "ifconfig lo:0 169.254.169.254 netmask 255.255.255.255 && ./limes start --config=/mnt/limes-conf/limes.conf $*"

