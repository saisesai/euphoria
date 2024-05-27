package euphoria

import (
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"time"
)

type HttpEventRetrieverConfig struct {
	EventEncode    string `yaml:"EventEncode"`
	BaseAddr       string `yaml:"BaseAddr"`
	IdleInterval   int    `yaml:"IdleInterval"`
	EventGetPath   string `yaml:"EventGetPath"`
	EventCountPath string `yaml:"EventCountPath"`
	EventClearPath string `yaml:"EventClearPath"`
}

type HttpEventRetriever struct {
	Config *HttpEventRetrieverConfig
	Logger *logrus.Entry
	Client *http.Client
	Next   EventQueue
}

func NewHttpEventRetriever(config *HttpEventRetrieverConfig, client *http.Client, next EventQueue) *HttpEventRetriever {
	return &HttpEventRetriever{
		Config: config,
		Logger: logrus.WithField("Fm", "HttpEventRetriever"),
		Client: client,
		Next:   next,
	}
}

func (m *HttpEventRetriever) Idle() {
	time.Sleep(time.Duration(m.Config.IdleInterval) * time.Millisecond)
}

func (m *HttpEventRetriever) Update() {
	// do request
	res, err := m.Client.Get(m.Config.BaseAddr + m.Config.EventGetPath)
	if err != nil {
		m.Logger.WithError(err).Errorln("failed to do get request!")
		m.Idle()
		return
	}
	defer res.Body.Close()
	// read data
	b, err := io.ReadAll(res.Body)
	if err != nil {
		m.Logger.WithError(err).Errorln("failed to read response data!")
		m.Idle()
		return
	}
	// decode data to event
	var events []*Event
	err = Decode(m.Config.EventEncode, b, &events)
	if err != nil {
		m.Logger.WithError(err).Errorln("failed to decode response data!")
		m.Idle()
		return
	}
	// process event
	m.Next.Lock()
	for _, event := range events {
		m.Next.Push(event)
		// TODO comment
		//if event.Nm == "TcpData" {
		//	fmt.Println("===rtv===")
		//	fmt.Println(string(event.Dt[:]))
		//}
	}
	m.Next.Unlock()
	// idle
	if len(events) == 0 {
		m.Idle()
	}
}

func (m *HttpEventRetriever) Run() {
	for {
		m.Update()
	}
}
