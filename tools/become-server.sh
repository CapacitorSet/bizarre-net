set -e

sudo nsenter --net=/var/run/netns/srvns sudo -u $(whoami) test 1
echo 'You are now in the server namespace.'
echo 'Server-client interface:'
sudo -E nsenter --net=/var/run/netns/srvns sudo -E -u $(whoami) ip addr show seth0
echo 'Server-host interface:'
sudo -E nsenter --net=/var/run/netns/srvns sudo -E -u $(whoami) ip addr show seth1
sudo -E nsenter --net=/var/run/netns/srvns sudo -E -u $(whoami) $SHELL
