package euphoria

import (
	"bytes"
	"github.com/sirupsen/logrus"
	"net/http"
	"time"
)

type HttpEventSenderConfig struct {
	EventEncode   string `yaml:"EventEncode"`
	BaseAddr      string `yaml:"BaseAddr"`
	IdleInterval  int    `yaml:"IdleInterval"`
	EventPostPath string `yaml:"EventPostPath"`
}

type HttpEventSender struct {
	EventQueueImpl
	Config *HttpEventSenderConfig
	Logger *logrus.Entry
	Client *http.Client
}

func NewHttpEventSender(config *HttpEventSenderConfig, client *http.Client) *HttpEventSender {
	return &HttpEventSender{
		EventQueueImpl: EventQueueImpl{},
		Config:         config,
		Logger:         logrus.WithField("From", "HttpEventSender"),
		Client:         client,
	}
}
func (m *HttpEventSender) Idle() {
	time.Sleep(time.Millisecond * time.Duration(m.Config.IdleInterval))
}

func (m *HttpEventSender) Run() {
	for {
		m.Lock()
		empty := m.Empty()
		m.Unlock()
		if empty {
			m.Idle()
			continue
		}
		// get event
		var events []*Event
		m.Lock()
		for !m.Empty() {
			events = append(events, m.Front())
			m.Pop()
		}
		m.Unlock()
		// send events
		// TODO post data type detect
		data, _ := Encode("json", &events)
		for {
			res, err := m.Client.Post(m.Config.BaseAddr+m.Config.EventPostPath,
				"application/json",
				bytes.NewReader(data[:]),
			)
			if err == nil && res.StatusCode == http.StatusOK {
				res.Body.Close()
				break
			}
			if err != nil {
				m.Logger.WithError(err).Errorln("failed to post events!")
			}
			if res.StatusCode != http.StatusOK {
				m.Logger.Errorf("unexpected status code: %v", res.StatusCode)
			}
			res.Body.Close()
		}
		//m.Logger.Debugf("send %v events", len(events))
	}
}
