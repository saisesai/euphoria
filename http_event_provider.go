package euphoria

import (
	"github.com/sirupsen/logrus"
	"net/http"
)

type HttpEventProviderConfig struct {
	EventEncode       string `yaml:"EventEncode"`
	HttpListenAddr    string `yaml:"HttpListenAddr"`
	BasePath          string `yaml:"BasePath"`
	EventGetPath      string `yaml:"EventGetPath"`
	EventCountPath    string `yaml:"EventCountPath"`
	EventClearPath    string `yaml:"EventClearPath"`
	MaxEventFetchSize int    `yaml:"MaxEventFetchSize"`
}

type HttpEventProvider struct {
	EventQueueImpl
	Config     *HttpEventProviderConfig
	Logger     *logrus.Entry
	HttpServer *http.ServeMux
}

func NewHttpEventProvider(config *HttpEventProviderConfig, httpServer *http.ServeMux) (provider *HttpEventProvider) {
	provider = &HttpEventProvider{
		EventQueueImpl: EventQueueImpl{},
		Config:         config,
		Logger:         logrus.WithField("Fm", "HttpEventProvider"),
		HttpServer:     httpServer,
	}
	provider.SetupHandler()
	return provider
}

func (m *HttpEventProvider) SetupHandler() {
	m.HttpServer.HandleFunc(m.Config.BasePath+m.Config.EventGetPath, m.HttpEventGetHandler())
	m.HttpServer.HandleFunc(m.Config.BasePath+m.Config.EventCountPath, m.HttpEventCountHandler())
	// TODO add clear
}

func (m *HttpEventProvider) HttpEventGetHandler() http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		// get events
		events := make([]*Event, 0, m.Config.MaxEventFetchSize)
		m.Lock()
		for !m.Empty() /*&& len(events) < m.Config.MaxEventFetchSize*/ {
			events = append(events, m.Front())
			m.Pop()
		}
		m.Unlock()
		// TODO comment
		//for _, event := range events {
		//	if event.Nm == "TcpData" {
		//		fmt.Println("===pv===")
		//		fmt.Println(string(event.Dt[:]))
		//	}
		//}
		// encode events
		bytes, err := Encode(m.Config.EventEncode, events)
		if err != nil {
			m.Logger.WithError(err).Errorln("invalid event encoding")
			return
		}
		// write events
		_, err = writer.Write(bytes[:])
		if err != nil {
			m.Logger.WithError(err).Errorln("failed to write data, do recovery!")
			m.Lock()
			m.Recovery(events[:])
			m.Unlock()
			return
		}
	}
}

func (m *HttpEventProvider) HttpEventCountHandler() http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		m.Lock()
		res := make(map[string]interface{})
		res["count"] = m.Count()
		b, _ := Encode(m.Config.EventEncode, &res)
		_, _ = writer.Write(b[:])
		m.Unlock()
	}
}
