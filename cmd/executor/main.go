package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strconv"

	"golang.org/x/crypto/ssh"
	co "vda.io/executor/pkg/config"
	ct "vda.io/executor/pkg/context"
	ho "vda.io/executor/pkg/host"
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

			signer, err := ssh.ParsePrivateKey(priv_key)
			if err != nil {
				log.Panicf("Failed to parse generated private key: %s", err)
			}
			client := connectToHost(host, host_key, signer)
			checkAgentVersion(client)
			runActions(client, host)
			if err = client.Close(); err != nil {
				log.Panicf("Failed closing SSH connection to %s: %s", host.Config.Hostname, err)
			}
		} else {
			log.Printf("  Host found in state: Starting connectivity check.")

			signer := context.GetSSHPrivateKey(host)
			client := connectToHost(host, host.State.HostKey, signer)
			checkAgentVersion(client)
			runActions(client, host)
			if err := client.Close(); err != nil {
				log.Panicf("Failed closing SSH connection to %s: %s", host.Config.Hostname, err)
			}
		}
	}
}

func connectToHost(host ho.Host, hostKey string, signer ssh.Signer) *ssh.Client {
	parsedHostKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(hostKey))
	if err != nil {
		log.Panicf("Failed to parse host key: %s", err)
	}

	slog.Debug("Parsed Host Key", "host_name", host.Name, "host_key", hostKey)

	sshConfig := ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.FixedHostKey(parsedHostKey),
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", host.Config.Hostname, host.Config.Port), &sshConfig)
	if err != nil {
		log.Panicf("Failed establishing SSH connection to %s: %s", host.Config.Hostname, err)
	}
	return client
}

func checkAgentVersion(client *ssh.Client) {
	_, stdout := utils.SSHRun(client, "./agent version")
	slog.Debug("Agent", "version", stdout)

	if stdout != commitHash {
		slog.Warn("Agent Version Mismatch", "local_version", commitHash, "remote_version", stdout)
	}
}

func runActions(client *ssh.Client, host ho.Host) {
	for actionName, actionArgs := range host.Config.Actions {
		switch actionName {
		case "copy":
			action := decodeActionArgs[co.CopyAction](actionName, actionArgs)
			runCopyAction(client, host, action)
		default:
			log.Fatalf("unknown action %q for host %q", actionName, host.Name)
		}
	}
}

func decodeActionArgs[T any](actionName string, actionArgs any) T {
	var decoded T
	raw, err := json.Marshal(actionArgs)
	if err != nil {
		log.Fatalf("failed to serialize %q action args: %s", actionName, err)
	}
	if err = json.Unmarshal(raw, &decoded); err != nil {
		log.Fatalf("failed to decode %q action args: %s", actionName, err)
	}
	return decoded
}

func runCopyAction(client *ssh.Client, host ho.Host, action co.CopyAction) {
	if action.Src == "" {
		log.Fatalf("copy action for host %q requires src", host.Name)
	}
	if action.Dst == "" {
		log.Fatalf("copy action for host %q requires dst", host.Name)
	}

	payload, err := os.ReadFile(action.Src)
	if err != nil {
		log.Fatalf("failed to read copy source %q for host %q: %s", action.Src, host.Name, err)
	}

	encoded := []byte(base64.StdEncoding.EncodeToString(payload))
	command := "./agent copy --filename " + strconv.Quote(action.Dst)
	exitCode, stdout := utils.SSHRunWithStdin(client, command, &encoded)
	if exitCode != 0 {
		log.Fatalf("copy action failed for host %q with exit code %d: %s", host.Name, exitCode, stdout)
	}
}
