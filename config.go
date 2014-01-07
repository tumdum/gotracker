package gotracker

type Network struct {
	Host string
	Port int
}

type Server struct {
	Interval       int
	DefaultNumWant int
	MaxNumWant     int
}

type Config struct {
	Network Network
	Server  Server
}
