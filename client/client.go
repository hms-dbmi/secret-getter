package client

// Client interface for clients to implements
type Client interface {
	Name() string
	//NewClient()
	List(path string) interface{}
	Read(path string) string
}
