package euphoria

import (
	"github.com/sirupsen/logrus"
	"io"
	"net"
	"strings"
	"sync"
	"time"
)

type TcpInputConfig struct {
	ListenAddr     string `yaml:"ListenAddr"`
	ReadBufferSize int    `yaml:"ReadBufferSize"`
	IdleInterval   int    `yaml:"IdleInterval"`
	OpenTimeout    int    `yaml:"OpenTimeout"`
}

type TcpInput struct {
	EventQueueImpl
	Config        *TcpInputConfig
	Logger        *logrus.Entry
	Listener      net.Listener
	Registry      map[string]*Connect
	RegistryMutex sync.RWMutex
	Next          EventQueue
}

func NewTcpInput(config *TcpInputConfig, next EventQueue) *TcpInput {
	// setup logger
	L := logrus.WithField("Fm", "NewTcpInput")
	// create instance
	tcpInput := &TcpInput{
		EventQueueImpl: EventQueueImpl{},
		Config:         config,
		Logger:         logrus.WithField("Fm", "TcpInput"),
		Listener:       nil,
		Registry:       make(map[string]*Connect),
		RegistryMutex:  sync.RWMutex{},
		Next:           next,
	}
	// create listener
	tcpListener, err := net.Listen("tcp", config.ListenAddr)
	if err != nil {
		L.WithError(err).Fatalln("failed to do tcp listen!")
	}
	tcpInput.Listener = tcpListener

	return tcpInput
}

func (m *TcpInput) Idle() {
	time.Sleep(time.Millisecond * time.Duration(m.Config.IdleInterval))
}

func (m *TcpInput) HandleConn(conn net.Conn) {
	// add conn to registry
	m.RegistryMutex.Lock()
	defer m.RegistryMutex.Unlock()
	connect := &Connect{
		Conn:  conn,
		From:  conn.RemoteAddr().String(),
		To:    "",
		Ready: make(chan bool),
	}
	m.Registry[conn.RemoteAddr().String()] = connect
	// log
	m.Logger.WithField("Alive", len(m.Registry)).Infof("conn %v connected!", conn.RemoteAddr())
	// send open event
	openEvent := &Event{
		Nm: "TcpOpen",
		To: "",
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

func (m *TcpInput) Poll(connect *Connect) {
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
		// log
		m.Logger.WithField("Alive", len(m.Registry)).Infof("conn %v closed!", connect.Conn.RemoteAddr().String())
	}()
	L := m.Logger.WithField("TcpFrom", connect.From).WithField("TcpTo", connect.To)
	// sync open event
	select {
	case <-time.After(time.Millisecond * time.Duration(m.Config.OpenTimeout)):
		m.Logger.WithField("ConnFrom", connect.From).Warn("open remote timeout!")
		return
	case <-connect.Ready:
		L.Debug("sync done!")
	}
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
		// send data event
		eventData := make([]byte, n)
		copy(eventData, buf[:n])
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

func (m *TcpInput) HandleEvents() {
	for {
		// check event queue
		m.Lock()
		empty := m.Empty()
		m.Unlock()
		if empty {
			m.Idle()
			continue
		}
		// get events
		var events []*Event
		m.Lock()
		for !m.Empty() {
			events = append(events, m.Front())
			m.Pop()
		}
		m.Unlock()
		// process events
		for _, event := range events {
			switch event.Nm {
			case "TcpOpen":
				m.HandleOpenEvent(event)
			case "TcpData":
				m.HandleDataEvent(event)
			case "TcpClose":
				m.HandleCloseEvent(event)
			default:
				m.Logger.Errorf("invalid event type: %v", event.Nm)
			}
		}
	}
}

func (m *TcpInput) HandleOpenEvent(event *Event) {
	L := m.Logger.WithField("TcpFrom", event.Fm).WithField("TcpTo", event.To)
	L.Debug("open event received!")
	m.RegistryMutex.Lock()
	defer m.RegistryMutex.Unlock()
	// check registry
	connect, exist := m.Registry[event.To]
	if !exist {
		L.Debug("cannot find conn in registry")
		return
	}
	// process event
	connect.To = event.Fm
	connect.Ready <- true
}

func (m *TcpInput) HandleDataEvent(event *Event) {
	L := m.Logger.WithField("TcpFrom", event.Fm).WithField("TcpTo", event.To)
	L.Debugf("data event received! data size: %v", len(event.Dt))
	m.RegistryMutex.Lock()
	// check registry
	connect, exist := m.Registry[event.To]
	m.RegistryMutex.Unlock()
	if !exist {
		L.Debug("cannot find conn in registry")
		return
	}
	// process event
	n, err := connect.Conn.Write(event.Dt[:])
	// TODO comment
	//fmt.Println("===ti===")
	//fmt.Println(string(event.Dt[:]))
	if err != nil {
		L.Error("failed to write conn!")
		return
	}
	L.Debugf("write %v bytes to conn!", n)
}

func (m *TcpInput) HandleCloseEvent(event *Event) {
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

func (m *TcpInput) Run() {
	m.Logger.Infof("listen at: %v", m.Config.ListenAddr)
	go m.HandleEvents()
	for {
		conn, err := m.Listener.Accept()
		if err != nil {
			m.Logger.WithError(err).Errorln("failed to accept conn!")
			continue
		}
		go m.HandleConn(conn)
	}
}
