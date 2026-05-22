package state

type State struct {
	Hosts map[string]HostState `json:"hosts"`
}

func NewState() State {
	return State{
		Hosts: map[string]HostState{},
	}
}

type HostState struct {
	HostKey    string `json:"host_key"`
	PrivateKey string `json:"private_key"`
}
