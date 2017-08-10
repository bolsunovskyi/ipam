package main

import (
	"flag"
	"log"

	"github.com/BurntSushi/toml"
	"github.com/docker/go-plugins-helpers/ipam"
	"gitlab.zeo.lcl/stopad/ipam/driver"
	"gopkg.in/go-playground/validator.v9"
)

var pluginName, configFile string
var verbose bool

type Config struct {
	GlobalPools []string `validate:"required"`
	LocalPools  []string `validate:"required"`
}

func init() {
	flag.StringVar(&pluginName, "n", "custom_ipam", "ipam driver name")
	flag.StringVar(&configFile, "c", "config.toml", "path to config file")
	flag.BoolVar(&verbose, "v", false, "verbose output")
	flag.Parse()
}

func main() {
	var conf Config
	if _, err := toml.DecodeFile(configFile, &conf); err != nil {
		log.Fatalln(err)
	}

	if err := validator.New().Struct(&conf); err != nil {
		log.Fatalln(err)
	}

	drv, err := driver.MakeIPAM(verbose, conf.GlobalPools, conf.LocalPools)
	if err != nil {
		log.Fatalln(err)
	}

	h := ipam.NewHandler(drv)
	log.Println("IPAM driver started...")
	if verbose {
		log.Println("Verbose output enabled")
	}
	if err := h.ServeTCP(pluginName, ":8080", "./", nil); err != nil {
		log.Fatalln(err)
	}
}
