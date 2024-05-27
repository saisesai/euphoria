package euphoria

import (
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
)

type HttpEventReceiverConfig struct {
	EventEncode    string `yaml:"EventEncode"`
	HttpListenAddr string `yaml:"HttpListenAddr"`
	BasePath       string `yaml:"BasePath"`
	EventPostPath  string `yaml:"EventPostPath"`
}

type HttpEventReceiver struct {
	Config     *HttpEventReceiverConfig
	Logger     *logrus.Entry
	HttpServer *http.ServeMux
	Next       EventQueue
}

func NewHttpEventReceiver(config *HttpEventReceiverConfig, httpServer *http.ServeMux, next EventQueue) *HttpEventReceiver {
	receiver := &HttpEventReceiver{
		Config:     config,
		Logger:     logrus.WithField("From", "HttpEventReceiver"),
		HttpServer: httpServer,
		Next:       next,
	}
	receiver.SetupHandler()
	return receiver
}

func (m *HttpEventReceiver) SetupHandler() {
	m.HttpServer.HandleFunc(m.Config.BasePath+m.Config.EventPostPath, m.HttpEventPostHandler())
}

func (m *HttpEventReceiver) HttpEventPostHandler() http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		// read data
		b, err := io.ReadAll(request.Body)
		if err != nil {
			m.Logger.WithError(err).Errorln("failed to read request data!")
			return
		}
		// make and send event
		var events []*Event
		err = Decode(m.Config.EventEncode, b, &events)
		if err != nil {
			m.Logger.WithError(err).Errorln("failed to decode events!")
			return
		}
		m.Next.Lock()
		for i := 0; i < len(events); i++ {
			m.Next.Push(events[i])
		}
		m.Next.Unlock()
	}
}
