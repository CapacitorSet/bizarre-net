sudo ip netns add srvns
sudo ip netns add clins
sudo ip link add seth0 type veth peer name ceth0
sudo ip link set seth0 netns srvns
sudo ip link set ceth0 netns clins
sudo nsenter --net=/var/run/netns/srvns sudo ip link set lo up
sudo nsenter --net=/var/run/netns/srvns sudo ip link set seth0 up
sudo nsenter --net=/var/run/netns/srvns sudo ip addr add 192.168.1.2/24 dev seth0
