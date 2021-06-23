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

The client will need to add a default route to the client TUN. Suppose that the server's *tunnel* IP (`ip addr show bizarre0`) is 10.0.10.1, then you must run the following:

```bash
# On the client
sudo ip route add default via 10.0.10.1 dev bizarre0 proto static
```

Furthermore, if the transport itself runs on a network interface (eg. Iodine) you must route the server IP (which is usually its public IP address) to the network interface.

For example, suppose that the Iodine server runs on 1.2.3.4, and that the client connects to the Internet via `eth0` with default gateway `192.168.0.1`. Then you must also run the following:

```bash
# On the client
sudo ip route add 1.2.3.4 via 192.168.0.1 dev eth0 proto static
```

You might need to enable local traffic on the interface (or both, if you're testing locally):

```bash
sudo sysctl -w net.ipv4.conf.bizarre0.accept_local=1
sudo sysctl -w net.ipv4.conf.bizarre1.accept_local=1
```

When troubleshooting you might want to ask yourself these questions:

 - Is the application using the correct route? For quick tests, you can use `curl --interface bizarre0`.
 - Are the packets being forwarded? Try sniffing the client TUN, the server TUN, and the Internet interface for the server.
 - Are the packets being masqueraded (i.e. do they come from the server IP and not from the tunnel IP when leaving the server)? Try sniffing the Internet interface for the server.

When developing, using network namespaces allows for multiple "clean" environments (no firewalls or iptables rules). `go run tools/setup/main.go` creates an environment with a "server" namespace `srvns` and a "client" namespace `clins`, connected by a virtual Ethernet cable `seth0-veth0`, respectively with the address 192.168.1.2/24 and 192.168.1.3/24. With `tools/become-server.sh` and `tools/become-client.sh` you can then open a shell into those namespaces. Finally, [here](https://iximiuz.com/en/posts/container-networking-is-simple/)'s a good tutorial about network namespaces.

## Testing
Set up the testing environment with `test/server-setup.sh` and `test/client-setup.sh`. Then:

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

## Licenses

This project is currently "all rights reserved", but will eventually get a proper license when it works.