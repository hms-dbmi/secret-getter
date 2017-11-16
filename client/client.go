package client

// Client interface for clients to implements
type Client interface {
	NewClient()
	List(path string) interface{}
	Read(path string) string
}
