package euphoria

import (
	"encoding/json"
	"github.com/sirupsen/logrus"
	"net/http"
)

type ServerConfig struct {
	Common struct {
		EventEncode    string `yaml:"EventEncode"`
		HttpListenAddr string `yaml:"HttpListenAddr"`
		BasePath       string `yaml:"BasePath"`
	} `yaml:"Common"`
	HttpEventProvider HttpEventProviderConfig `yaml:"HttpEventProvider"`
	HttpEventReceiver HttpEventReceiverConfig `yaml:"HttpEventReceiver"`
	TcpOutput         TcpOutputConfig         `yaml:"TcpOutput"`
}

func (m *ServerConfig) String() string {
	b, _ := json.MarshalIndent(m, "", "\t")
	return string(b)
}

type Server struct {
	Config            *ServerConfig
	Logger            *logrus.Entry
	HttpServer        *http.ServeMux
	HttpEventProvider *HttpEventProvider
	HttpEventReceiver *HttpEventReceiver
	TcpOutput         *TcpOutput
}

func NewServer(config *ServerConfig) (server *Server) {
	server = &Server{
		Config:     config,
		Logger:     logrus.WithField("Fm", "HttpServer"),
		HttpServer: http.NewServeMux(),
	}
	server.HttpEventProvider = NewHttpEventProvider(&config.HttpEventProvider, server.HttpServer)
	server.TcpOutput = NewTcpOutput(&config.TcpOutput, server.HttpEventProvider)
	server.HttpEventReceiver = NewHttpEventReceiver(&config.HttpEventReceiver, server.HttpServer, server.TcpOutput)
	return server
}

func (m *Server) Run() {
	go m.TcpOutput.Run()
	m.Logger.Info("listen at:", m.Config.Common.HttpListenAddr)
	_ = http.ListenAndServe(m.Config.Common.HttpListenAddr, m.HttpServer)
}
