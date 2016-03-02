#!/bin/bash -eu
# Launches limes inside the docker container. Use this command to communicate with the daemon.
# Example usage: ./docker-run.sh profile dev
docker exec -it limes-daemon ./limes $*

