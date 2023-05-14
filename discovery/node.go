package discovery

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

var schemeName string

type Option struct {
	Endpoints []string
	Prefix    string
	TTL       int64
}

type Node struct {
	Id   string `json:"id"`
	Name string `json:"name"`
	Host string `json:"host"`
	Port int    `json:"port"`
}

func NewNode(name, host string, port int) *Node {
	return &Node{
		Id:   uuid.New().String(),
		Name: name,
		Host: host,
		Port: port,
	}
}

func (n *Node) Path(prefix string) string {
	return fmt.Sprintf("/%s/%s", prefix, n.Name)
}

func (n *Node) Key(prefix string) string {
	return fmt.Sprintf("%s/%s", n.Path(prefix), n.Id)
}

func (n *Node) Encode() string {
	data, _ := json.Marshal(n)
	return string(data)
}

func (n *Node) Decode(value []byte) error {
	if err := json.Unmarshal(value, &n); err != nil {
		return err
	}

	return nil
}
