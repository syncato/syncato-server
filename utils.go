package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/jmcvetta/randutil"
	"github.com/syncato/syncato-lib/config"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Options struct {
	createconfig bool

	port      int
	config    string
	loglevel  int
	logformat string
	httplog   string
	applog    string
}

func getServerOptions() (*Options, error) {
	opts := Options{}

	flag.BoolVar(&opts.createconfig, "createconfig", false, "Creates a custom configuration file")

	flag.IntVar(&opts.port, "port", 8080, "Listening port for HTTP/HTTPS server")
	flag.StringVar(&opts.config, "config", "config.json", "Configuration file")
	flag.IntVar(&opts.loglevel, "loglevel", 4, "Log level. Possible values: 0=panic,1=fatal,2=error,3=warning,4=info,5=debug")
	flag.StringVar(&opts.logformat, "logformat", "text", "Log format. Possible values: text or json")
	flag.StringVar(&opts.applog, "applog", "stdin", "File to output logs generated by the app")
	flag.StringVar(&opts.httplog, "reqlog", "stdin", "File to output requests Apache-like logs")

	flag.Parse()

	return &opts, nil
}

func createConfigFile(cp *config.ConfigProvider) error {
	reader := bufio.NewReader(os.Stdin)

	// Ask for the port
	portOK := false
	var port uint64 = DEFAULT_PORT
	for portOK == false {
		fmt.Printf("In which port the server is going to listen ? (%d) : ", DEFAULT_PORT)
		portText, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		portText = strings.TrimSuffix(portText, "\n")
		if portText == "" {
			port = DEFAULT_PORT
		} else {
			port, err = strconv.ParseUint(portText, 10, 64)
			if err != nil {
				fmt.Println("Error: port must be a number")
			}
		}
		portOK = true
	}

	// Ask the location to save the configuration
	fmt.Printf("Where do you want to save the config file ? (%s) : ", DEFAULT_CONFIG_NAME)
	configFilename, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	configFilename = strings.TrimSuffix(configFilename, "\n")
	if configFilename == "" {
		configFilename = DEFAULT_CONFIG_NAME
	}
	configFilename = filepath.Clean(configFilename)

	// create random secret for signing tokens
	secret, err := randutil.AlphaString(60)
	if err != nil {
		return err
	}

	var cfg = &config.Config{}
	cfg.Port = int(port)
	cfg.TokenSecret = secret
	cfg.TokenCipherSuite = DEFAULT_TOKEN_CIPHER_SUITE

	err = cp.CreateConfig(cfg, configFilename)
	if err != nil {
		return err
	}

	fmt.Println("Configuration created succesfully!")
	return nil
}
