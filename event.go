package euphoria

type Event struct {
	Nm string // Name
	To string // To
	Fm string // From
	Tm int64  // Time
	Dt []byte // Data
}
