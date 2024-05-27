package euphoria

type Event struct {
	Name string
	To   string
	From string
	Time int64
	Data []byte
}
