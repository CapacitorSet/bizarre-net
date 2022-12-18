# bizarre-net

A project to run Internet connections over bizarre media. Work in progress.

## Instructions

Configure `client/config.toml` and `server/config.toml` as needed, then:

```bash
cd server
go build
sudo setcap CAP_NET_ADMIN+ep ./server
cd ../client
go build
sudo setcap CAP_NET_ADMIN+ep ./client

./server # On the server machine
./client # On the client machine
```

## Tips

To enable forwarding on the server and thus be able to connect to the Internet (where `bizarre0` is the server TUN and `eth0` is an interface connected to the Internet):

```bash
# On the server
sudo sysctl -w net.ipv4.ip_forward=1
sudo iptables -t filter -I FORWARD -i bizarre0 -o eth0 -j ACCEPT
sudo iptables -t filter -I FORWARD -m state -i bizarre0 --state ESTABLISHED,RELATED -j ACCEPT
sudo iptables -t nat -I POSTROUTING -o eth0 -j MASQUERADE
```

You might need to enable local traffic on the interface (or both, if you're testing locally):

```bash
sudo sysctl -w net.ipv4.conf.bizarre0.accept_local=1
sudo sysctl -w net.ipv4.conf.bizarre1.accept_local=1
```

When troubleshooting you might want to ask yourself these questions:

 - Is the application using the correct route? `ip route get <ip>` is your friend.
 - Are the packets being forwarded? Try sniffing the client TUN, the server TUN, and the Internet interface for the server.
 - Are the packets being masqueraded (i.e. do they come from the server IP and not from the tunnel IP when leaving the server)? Try sniffing the Internet interface for the server.

When developing, using network namespaces allows for multiple "clean" environments (no firewalls or iptables rules). `go run tools/setup/main.go` creates an environment with a "server" namespace `srvns` and a "client" namespace `clins`, connected by a virtual Ethernet cable `seth0-veth0`, respectively with the address 192.168.1.2/24 and 192.168.1.3/24. With `tools/become-server.sh` and `tools/become-client.sh` you can then open a shell into those namespaces. Finally, [here](https://iximiuz.com/en/posts/container-networking-is-simple/)'s a good tutorial about network namespaces.

## Testing
Set up the testing environment with `go run tools/setup/main.go`. Then:

```bash
# First run this in a terminal
tools/become-server.sh
cd test/udp
sudo go test -run TestUDPServer
```

```bash
# Then run this in another terminal
tools/become-client.sh
cd test/udp
sudo go test -run TestUDPClient
```

`sudo go run run_tests.go` is also available, but it is meant for quick tests where you're not interested in inspecting the output and for continuous integration.

## Todo list

[x] Command execution
[ ] File upload/download/exploration
[ ] Rootless mode (disables TUN creation)
[ ] Password authentication
[ ] Compression
[ ] ICMP transport
[ ] DNS transport
[ ] Version compatibility check (embed in hello message)
[ ] Write tests
[ ] Test IPv6 support
[ ] Testing on Windows

## Licenses

This project is currently "all rights reserved", but will eventually get a proper license when it works.