package host

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"log"
	"log/slog"
	"net"

	"golang.org/x/crypto/ssh"
	co "vda.io/executor/pkg/config"
	st "vda.io/executor/pkg/state"
	"vda.io/executor/pkg/utils"
)

type Host struct {
	Name   string
	Config *co.HostConfig
	State  *st.HostState
}

func (h *Host) getHostKey() (*ssh.Client, string) {
	var host_key ssh.PublicKey
	ssh_config := ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{
			ssh.Password("root"),
		},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			host_key = key
			return nil
		},
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", h.Config.Hostname, h.Config.Port), &ssh_config)
	if err != nil {
		log.Fatal(err)
	}

	return client, string(ssh.MarshalAuthorizedKey(host_key))
}

func (h *Host) genNewSSHKeys() ([]byte, []byte) {
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		log.Fatal(err)
	}
	mPrivKey, err := ssh.MarshalPrivateKey(privKey, "")
	if err != nil {
		log.Fatal(err)
	}
	sshPubKey, err := ssh.NewPublicKey(pubKey)
	if err != nil {
		log.Fatal(err)
	}
	return pem.EncodeToMemory(mPrivKey), ssh.MarshalAuthorizedKey(sshPubKey)
}

func (h *Host) Bootstrap(agent []byte) (string, []byte) {
	fmt.Printf("        Getting host key: ")
	client, host_key := h.getHostKey()
	fmt.Printf("OK\n")

	slog.Debug("getHostKey", "host_key", host_key)

	fmt.Printf("        Getting new SSH keys: ")
	priv_key, pub_key := h.genNewSSHKeys()
	fmt.Printf("OK\n")

	fmt.Printf("        Installing SSH public key: ")
	utils.SSHRun(client, fmt.Sprintf("echo '%s' >> ~/.ssh/authorized_keys", string(pub_key)))
	fmt.Printf("OK\n")

	fmt.Printf("        Installing agent: ")
	exit_code, stdout := utils.InstallAgent(client, agent)
	if exit_code != 0 {
		log.Fatalf("Failed to install agent with exit code %d: %s", exit_code, stdout)
	}
	fmt.Printf("OK\n")

	return host_key, priv_key
}
