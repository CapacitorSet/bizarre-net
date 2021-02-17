sudo nsenter --net=/var/run/netns/clins sudo ip link set lo up
sudo nsenter --net=/var/run/netns/clins sudo ip link set ceth0 up
sudo nsenter --net=/var/run/netns/clins sudo ip addr add 192.168.1.3/24 dev ceth0
