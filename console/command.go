package console

import (
	"fmt"
	"strings"

	"sc.y3/dispatcher"
)

var (
	disp *dispatcher.Client
)

type Command struct {
	operation string
	args      []string
}

func Parse(raw string) Command {
	ret := Command{"", make([]string, 0)}

	for i, v := range strings.Split(raw, " ") {
		switch i {
		case 0:
			ret.operation = v
		default:
			ret.args = append(ret.args, v)
		}
	}

	return ret
}

func (c *Command) Exec() (string, error) {
	switch c.operation {
	case "state":
		return State(c.args[0])
	case "list":
		return ListPeers(c.args[0])
	case "log":
		return Log(c.args[0])
	case "update":
		return UpdateState(c.args[0], c.args[1], c.args[2])
	case "leader":
		return Leader(c.args[0])
	default:
		return "", fmt.Errorf("No such command")
	}
}
