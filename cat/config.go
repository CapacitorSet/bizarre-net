package cat

type clientConfig struct {
	ServerName string // The name of the person that will receive packets
	ClientName string // The name of the person that will send packets
}

type serverConfig struct {
	ServerName string // The name of the person that will receive packets
}
