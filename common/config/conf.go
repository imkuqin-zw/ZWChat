package config


type Server struct {
	Proto string
	Addr  string
}

type Path struct {
	Root string
}

type Log struct {
	Alsologtostderr string
	LogDir string
}