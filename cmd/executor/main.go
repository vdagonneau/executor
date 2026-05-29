package main

import (
	"fmt"
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
var commitHash string

func main() {
	opts := &slog.HandlerOptions{
		Level: utils.GetLogLevelFromEnv(),
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, opts)))

	context := ct.NewContext()

	fmt.Printf("Checking Hosts\n")
	for host_name, host := range context.Hosts {
		if host.State == nil {
			fmt.Printf("  %s: Host not found in state.\n", host_name)

			fmt.Printf("      ⬇️ Starting bootstrap.\n")
			host_key, priv_key := host.Bootstrap(agent)
			fmt.Printf("      ⬆️ Bootstrap complete!\n")

			fmt.Printf("      💾 Saving host state: ")
			encrypted_priv_key := context.Encrypt(priv_key)
			context.State.Hosts[host_name] = st.HostState{HostKey: host_key, PrivateKey: encrypted_priv_key}
			context.SaveState()
			fmt.Printf("✅\n")
		} else {
			log.Printf("  Host found in state: Starting connectivity check.")

			host_key, _, _, _, err := ssh.ParseAuthorizedKey([]byte(host.State.HostKey))
			if err != nil {
				log.Panicf("Failed to parse host key: %s", err)
			}

			slog.Debug("Parsed Host Key", "host_name", host_name, "host_key", host.State.HostKey)

			signer := context.GetSSHPrivateKey(host)

			ssh_config := ssh.ClientConfig{
				User: "root",
				Auth: []ssh.AuthMethod{
					ssh.PublicKeys(signer),
				},
				HostKeyCallback: ssh.FixedHostKey(host_key),
			}

			client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", host.Config.Hostname, host.Config.Port), &ssh_config)
			if err != nil {
				log.Panicf("Failed establishing SSH connection to %s: %s", host.Config.Hostname, err)
			}

			_, stdout := utils.SSHRun(client, "./agent --version")
			slog.Debug("Agent", "version", stdout)

			if stdout != commitHash {
				slog.Warn("Agent Version Mismatch", "local_version", commitHash, "remote_version", stdout)
			}
		}
	}
}
