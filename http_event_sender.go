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
		Logger:         logrus.WithField("Fm", "HttpEventSender"),
		Client:         client,
	}
}
func (m *HttpEventSender) Idle() {
	time.Sleep(time.Millisecond * time.Duration(m.Config.IdleInterval))
}

func (m *HttpEventSender) Update() {
	m.Lock()
	empty := m.Empty()
	m.Unlock()
	if empty {
		m.Idle()
		return
	}
	// get event
	var events []*Event
	m.Lock()
	for !m.Empty() {
		events = append(events, m.Front())
		m.Pop()
	}
	m.Unlock()

	// TODO post data type detect
	data, err := Encode(m.Config.EventEncode, &events)
	if err != nil {
		m.Logger.WithError(err).Error("")
		return
	}
	// send events
	for {
		res, err := m.Client.Post(m.Config.BaseAddr+m.Config.EventPostPath,
			m.Config.EventEncode,
			bytes.NewReader(data[:]),
		)
		res.Body.Close()
		if err == nil && res.StatusCode == http.StatusOK {
			break
		} else {
			m.Logger.WithError(err).WithField("Code", res.StatusCode).Errorln("failed to post events! retry!")
		}
	}
	//m.Logger.Debugf("send %v events", len(events))
}

func (m *HttpEventSender) Run() {
	for {
		m.Update()
	}
}
