package sgin

type Error struct {
	Code int
	Err  error
	Meta any
}

func (msg Error) Error() string {
	return msg.Err.Error()
}
