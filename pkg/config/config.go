package config

type Config struct {
	AgeIdentities string                `json:"age_identities"`
	Hosts         map[string]HostConfig `json:"hosts"`
}

type HostConfig struct {
	Hostname string `json:"hostname"`
	Port     int    `json:"port"`
}
