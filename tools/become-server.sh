sudo nsenter --net=/var/run/netns/srvns sudo -u $(whoami) $SHELL
