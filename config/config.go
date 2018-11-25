package config

import (
	"flag"
	"net"
)

//Config arguments passed to program
type Config struct {
	Ip         string
	Port       string
	Introducer string
	Prod       bool
	IsIntro    bool
	FalseType  int
	// FileStoragePos string
}

// false positive rate is 0, 3% , 10% or 30%

//SetupFlags handles incoming arguments and storeds them in the Config struct
func SetupFlags() Config {
	config := new(Config)
	flag.StringVar(&config.Ip, "ip", getIP(), "specify the ip address of self")
	flag.StringVar(&config.Port, "port", "9123", "The port the service will run on. Default is 9123.")
	flag.StringVar(&config.Introducer, "introducer", config.Ip+":"+config.Port, "IP adress to introducer of group")
	flag.BoolVar(&config.IsIntro, "isIntro", false, "If a node is an introducer or not")
	flag.BoolVar(&config.Prod, "prod", false, "running the system in production")
	flag.IntVar(&config.FalseType, "falserate", 0, "The false positive rate type, default is no false positive")
	flag.Parse()

	if config.Prod {
		config.Introducer = "fa18-cs425-g14-01.cs.illinois.edu:9123"
	}

	return *config
}

func getIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return "127.0.0.1"
}
