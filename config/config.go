package config

type Node struct {
	// full URL https://user:pw@domain.com
	Id     string `json:"id"`
	Server string `json:"server"`
	// name of the session cookie
	Cookie             string `json:"cookie"`
	MaxConnections     int    `json:"maxConnections"`
	InsecureSkipVerify bool   `json:"insecureSkipVerify"`
}

type Variant struct {
	Id string `json:"id"`
	// 0-100 %
	Share int64   `json:"share"`
	Nodes []*Node `json:"nodes"`
}

type Config struct {
	Id             string     `json:"id"`
	Variants       []*Variant `json:"variants"`
	SessionTimeout int        `json:"sessionTimeout"`
}
