package main

import (
	"flag"
	"fmt"
	"github.com/elazarl/goproxy"
	"github.com/saisesai/euphoria"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"net/http"
	"os"
)

var argDebug = flag.Bool("debug", false, "debug mode")
var argMode = flag.String("mode", "", "running mode: [client/server/http-proxy]")
var argConfig = flag.String("config", "", "config file path")

var L = logrus.WithField("From", "main")

func processArgs() {
	flag.Parse()
	var argErr = false
	if *argMode != "client" && *argMode != "server" && *argMode != "http-proxy" {
		L.Errorln("wrong arg [mode]!")
		argErr = true
	}
	if *argConfig == "" {
		L.Errorln("arg [config] is required!")
		argErr = true
	}
	if argErr {
		flag.PrintDefaults()
		os.Exit(-1)
	}
}

func main() {
	if *argDebug {
		logrus.SetLevel(logrus.DebugLevel)
	} else {
		logrus.SetLevel(logrus.InfoLevel)
	}
	processArgs()
	if *argMode == "server" {
		config := &euphoria.ServerConfig{}
		b, err := os.ReadFile(*argConfig)
		if err != nil {
			L.WithError(err).Fatalln("failed to read config file!")
		}
		err = yaml.Unmarshal(b, config)
		if err != nil {
			L.WithError(err).Fatalln("failed to decode config file!")
		}
		fmt.Println(config)
		euphoria.NewServer(config).Run()
	}
	if *argMode == "client" {
		config := &euphoria.ClientConfig{}
		b, err := os.ReadFile(*argConfig)
		if err != nil {
			L.WithError(err).Fatalln("failed to read config file!")
		}
		err = yaml.Unmarshal(b, config)
		if err != nil {
			L.WithError(err).Fatalln("failed to decode config file!")
		}
		fmt.Println(config)
		euphoria.NewClient(config).Run()
	}
	if *argMode == "http-proxy" {
		L.Info("http proxy listen at localhost:3003")
		proxy := goproxy.NewProxyHttpServer()
		if *argDebug {
			proxy.Verbose = true
		}
		L.Fatal(http.ListenAndServe("localhost:3003", proxy))
	}
}
