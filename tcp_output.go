package euphoria

import (
	"github.com/sirupsen/logrus"
	"io"
	"net"
	"strings"
	"sync"
	"time"
)

type TcpOutputConfig struct {
	DestAddr       string `yaml:"DestAddr"`
	IdleInterval   int    `yaml:"IdleInterval"`
	ReadBufferSize int    `yaml:"ReadBufferSize"`
}

type TcpOutput struct {
	EventQueueImpl
	Config        *TcpOutputConfig
	Logger        *logrus.Entry
	Registry      map[string]*Connect
	RegistryMutex sync.Mutex
	Next          EventQueue
}

func NewTcpOutput(config *TcpOutputConfig, next EventQueue) *TcpOutput {
	tcpOutput := &TcpOutput{
		EventQueueImpl: EventQueueImpl{},
		Config:         config,
		Logger:         logrus.WithField("Fm", "TcpOutput"),
		Registry:       make(map[string]*Connect),
		RegistryMutex:  sync.Mutex{},
		Next:           next,
	}
	return tcpOutput
}

func (m *TcpOutput) Idle() {
	time.Sleep(time.Millisecond * time.Duration(m.Config.IdleInterval))
}

func (m *TcpOutput) Update() {
	// check event queue
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
	// process events
	for i := 0; i < len(events); i++ {
		switch events[i].Nm {
		case "TcpOpen":
			m.HandleOpenEvent(events[i])
		case "TcpData":
			m.HandleDataEvent(events[i])
		case "TcpClose":
			m.HandleCloseEvent(events[i])
		default:
			m.Logger.Errorf("invalid event type: %v", events[i].Nm)
		}
	}
}

func (m *TcpOutput) Run() {
	for {
		m.Update()
	}
}

func (m *TcpOutput) Poll(connect *Connect) {
	defer func() {
		connect.Conn.Close()
		// remove from registry
		m.RegistryMutex.Lock()
		defer m.RegistryMutex.Unlock()
		delete(m.Registry, connect.From)
		m.Logger.Debugf("conn %v removed from registry", connect.From)
		// send close event
		closeEvent := &Event{
			Nm: "TcpClose",
			To: connect.To,
			Fm: connect.From,
			Tm: time.Now().UnixNano(),
			Dt: nil,
		}
		m.Next.Push(closeEvent)
		m.Logger.WithField("Alive", len(m.Registry)).Infof("conn %v closed!", connect.To)
	}()
	// poll
	buf := make([]byte, m.Config.ReadBufferSize)
	for {
		// read data
		n, err := connect.Conn.Read(buf[:])
		if err != nil {
			if err != io.EOF && !strings.Contains(err.Error(), "use of closed network connection") {
				m.Logger.WithError(err).Errorln("failed to read conn!")
			}
			break
		}
		m.Logger.Debugf("read %v bytes form %v", n, connect.Conn.RemoteAddr().String())
		eventData := make([]byte, n)
		copy(eventData, buf[:n])
		// send data event
		dataEvent := &Event{
			Nm: "TcpData",
			To: connect.To,
			Fm: connect.From,
			Tm: time.Now().UnixNano(),
			Dt: eventData,
		}
		m.Next.Lock()
		m.Next.Push(dataEvent)
		m.Next.Unlock()
	}
}

func (m *TcpOutput) HandleOpenEvent(event *Event) {
	L := m.Logger.WithField("TcpFrom", event.Fm).WithField("TcpTo", event.To)
	L.Debug("open event received!")
	// dial to dest
	conn, err := net.Dial("tcp", m.Config.DestAddr)
	if err != nil {
		m.Logger.WithError(err).Error("failed to dial to dest!")
		return
	}
	// make conn and add it to registry
	connect := &Connect{
		Conn:  conn,
		From:  conn.LocalAddr().String(),
		To:    event.Fm,
		Ready: nil,
	}
	m.RegistryMutex.Lock()
	m.Registry[connect.From] = connect
	m.RegistryMutex.Unlock()
	// log
	m.Logger.WithField("Alive", len(m.Registry)).Infof("conn %v connected!", event.Fm)
	// make open event back to origin
	openEvent := &Event{
		Nm: "TcpOpen",
		To: connect.To,
		Fm: connect.From,
		Tm: time.Now().UnixNano(),
		Dt: nil,
	}
	m.Next.Lock()
	m.Next.Push(openEvent)
	m.Next.Unlock()
	// poll conn
	go m.Poll(connect)
}

func (m *TcpOutput) HandleDataEvent(event *Event) {
	L := m.Logger.WithField("TcpFrom", event.Fm).WithField("TcpTo", event.To)
	L.Debugf("data event received! data size: %v", len(event.Dt))
	m.RegistryMutex.Lock()
	// check registry
	connect, exist := m.Registry[event.To]
	m.RegistryMutex.Unlock()
	if !exist {
		L.Debug("cannot find conn in registry!")
		return
	}
	// process event
	_, _ = connect.Conn.Write(event.Dt[:])
}

func (m *TcpOutput) HandleCloseEvent(event *Event) {
	L := m.Logger.WithField("TcpFrom", event.Fm).WithField("TcpTo", event.To)
	L.Debug("close event received!")
	m.RegistryMutex.Lock()
	defer m.RegistryMutex.Unlock()
	// check registry
	connect, exist := m.Registry[event.To]
	if !exist {
		L.Debug("cannot find conn in registry")
		return
	}
	// process event
	connect.Conn.Close()
}
