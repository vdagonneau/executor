package main

import (
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"

	"golang.org/x/crypto/ssh"
	ct "vda.io/executor/pkg/context"
	st "vda.io/executor/pkg/state"
	"vda.io/executor/pkg/utils"

	_ "embed"
)

//go:embed embed/agent
var agent []byte

func main() {
	opts := &slog.HandlerOptions{
		Level: utils.GetLogLevelFromEnv(),
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, opts)))

	context := ct.NewContext()

	log.Printf("Checking Hosts")
	for host_name, host := range context.Hosts {
		if host.State == nil {
			log.Printf("  Host not found in state: Starting bootstrap.")
			host_key, priv_key := host.Bootstrap(agent)

			encrypted_priv_key := context.Encrypt(priv_key)
			context.State.Hosts[host_name] = st.HostState{HostKey: host_key, PrivateKey: encrypted_priv_key}
			context.SaveState()
		} else {
			log.Printf("  Host found in state: Starting connectivity check.")

			host_key, _, _, _, err := ssh.ParseAuthorizedKey([]byte(host_state.HostKey))
			if err != nil {
				log.Panicf("Failed to parse host key: %s", err)
			}

			slog.Debug("Parsed Host Key", "host_name", host_name, "host_key", host_state.HostKey)

			ssh_config := ssh.ClientConfig{
				User: "root",
				Auth: []ssh.AuthMethod{
					ssh.PublicKeys(signer),
				},
				HostKeyCallback: ssh.FixedHostKey(host_key),
			}

			client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", host.Hostname, host.Port), &ssh_config)
			if err != nil {
				log.Panicf("Failed establishing SSH connection to %s: %s", host.Hostname, err)
			}

			session, err := client.NewSession()
			stdout, err := session.StdoutPipe()
			if err != nil {
				log.Panicf("%s", err)
			}
			session.Run("./agent --version")

			out, err := io.ReadAll(stdout)
			if err != nil {
				log.Panicf("%s", err)
			}
			slog.Debug("Agent", "version", string(out))
		}
	}
}
