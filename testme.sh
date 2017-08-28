IMG=8c84ff8147a3
docker save $IMG | sudo ~/bin/docker-squash -t squashed| docker load
