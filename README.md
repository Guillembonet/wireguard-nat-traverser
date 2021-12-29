# Wireguard NAT traverser

## How to build

Install golang and use `go build ./cmd/client` and `go build ./cmd/server` on the project root to generate the executables. Then try to run them without arguments to see usage.

## How does it work?

To understand how it works we will look at the different available commands:

### add <host_id>

Creates a wireguard interface on the client and assigns it an ip of 10.1.0.<host_id>, then the client sends a message to the server using UDP which contains the public key of the wireguard interface and the host id. The server will add a peer in wireguard which that public key and with allowed IP of 10.1.0.<host_id>. The server will then reply with the same message and the client will do exactly the same but with the public key it receives (the server's) and allowed IP of 10.1.0.1 but also adding the known endpoint of the server, as the port is well known.

Now the client and the server will have established a wireguard tunnel so once it is formed they will use it to send each other messages instead of using plain connection to improve security.

### connect <public_key>

The client sends a message to the server with `get <public_key>`, the server receives the message and searches in its own wireguard peers if some has that public key, if some peer has it, it returns that peer's IP and endpoint. The client then adds this peer to its wireguard configuration.

If a client does this it is able to add a peer which is behind a restrictive NAT because that NAT is being traversed in the connection between the server and that peer. The peer will need to do the same in order to form a connection to the first client, then they will both exploit the known address translation to form a wireguard tunnel between them.

### remove

Removes the server from the peer list of the client, the server is not needed anymore once the connection is established.

### set consumer|provider

Sets the client's role. Under the hood, setting it as consumer means that a new received peer will have allowed ip set to 0.0.0.0/0 which is every address, in this way we will route all the traffic to the other peer. It also applies linux routing commands recommended in [the wireguard website](https://www.wireguard.com/netns/#improved-rule-based-routing). If the role is set to provider which is the default it doesn't do anything, but to make it work we should run commands for applying routing to iptables and allowing ipv4 forwarding as explained in [this article](https://medium.com/tangram-visions/what-they-dont-tell-you-about-setting-up-a-wireguard-vpn-46f7bd168478).

### exit

Closes the program and reverts the routing and interfaces created.

## Conclusion

This project serves as a PoC for a simple program which allows 2 peers which are behind restrictive NATS to establish a wireguard tunnel with the help of another peer which has a less restrictive NAT, open ports or a public IP.
