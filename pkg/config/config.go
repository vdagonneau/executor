package config

type Config struct {
	AgeIdentities string                `json:"age_identities"`
	Hosts         map[string]HostConfig `json:"hosts"`
}

type HostConfig struct {
	Hostname string         `json:"hostname"`
	Port     int            `json:"port"`
	Actions  map[string]any `json:"actions"`
}

type CopyAction struct {
	Src string `json:"src"`
	Dst string `json:"dst"`
}
