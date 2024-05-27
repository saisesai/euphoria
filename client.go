package euphoria

import (
	"encoding/json"
	"github.com/sirupsen/logrus"
	"net/http"
)

type ClientConfig struct {
	Common struct {
		EventEncode string `yaml:"EventEncode"`
		BaseAddr    string `yaml:"BaseAddr"`
	} `yaml:"Common"`
	TcpInput           TcpInputConfig           `yaml:"TcpInput"`
	HttpEventRetriever HttpEventRetrieverConfig `yaml:"HttpEventRetriever"`
	HttpEventSender    HttpEventSenderConfig    `yaml:"HttpEventSender"`
}

func (m *ClientConfig) String() string {
	b, _ := json.MarshalIndent(m, "", "\t")
	return string(b)
}

type Client struct {
	Config             *ClientConfig
	Logger             *logrus.Entry
	HttpClient         *http.Client
	TcpInput           *TcpInput
	HttpEventRetriever *HttpEventRetriever
	HttpEventSender    *HttpEventSender
}

func NewClient(config *ClientConfig) (client *Client) {
	client = &Client{
		Config:     config,
		Logger:     logrus.WithField("From", "Client"),
		HttpClient: &http.Client{},
	}
	client.HttpEventSender = NewHttpEventSender(&config.HttpEventSender, client.HttpClient)
	client.TcpInput = NewTcpInput(&config.TcpInput, client.HttpEventSender)
	client.HttpEventRetriever = NewHttpEventRetriever(&config.HttpEventRetriever, client.HttpClient, client.TcpInput)
	return client
}

func (m *Client) Run() {
	go m.HttpEventSender.Run()
	go m.TcpInput.Run()
	m.HttpEventRetriever.Run()
}
