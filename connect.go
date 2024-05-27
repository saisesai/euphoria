package euphoria

import (
	"net"
)

type Connect struct {
	Conn  net.Conn
	From  string
	To    string
	Ready chan bool
}
