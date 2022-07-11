set -e

sudo nsenter --net=/var/run/netns/clins sudo -u $(whoami) test 1
echo 'You are now in the client namespace.'
echo 'Client interface:'
sudo -E nsenter --net=/var/run/netns/clins sudo -E -u $(whoami) ip addr show ceth0
sudo -E nsenter --net=/var/run/netns/clins sudo -E -u $(whoami) $SHELL
