package mock

type Writer struct {
	WriteFunc func(p []byte) (n int, err error)
}

func (m Writer) Write(p []byte) (n int, err error) {
	return m.WriteFunc(p)
}
