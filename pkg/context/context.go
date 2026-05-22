package context

import (
	"bytes"
	"encoding/base64"
	"errors"
	"io"
	"log"
	"log/slog"
	"os"
	"os/user"
	"path/filepath"

	"filippo.io/age"
	"golang.org/x/crypto/ssh"
	co "vda.io/executor/pkg/config"
	ho "vda.io/executor/pkg/host"
	st "vda.io/executor/pkg/state"
	"vda.io/executor/pkg/utils"
)

type Context struct {
	WorkingDir  string
	CurrentUser *user.User
	Identities  []age.Identity
	Recipients  []age.Recipient
	Config      co.Config
	StatePath   string
	State       st.State
	Hosts       map[string]ho.Host
}

func NewContext() Context {
	working_dir, err := os.Getwd()
	if err != nil {
		log.Panicf("Failed to get current working dir: %s", err)
	}
	slog.Debug("Get Working Directory", "working_dir", working_dir)

	current_user, err := user.Current()
	if err != nil {
		log.Panicf("Failed to get current user: %s", err)
	}
	slog.Debug("Get Current User", "user", current_user.Username, "home", current_user.HomeDir)

	var config co.Config
	config_path := filepath.Join(working_dir, "config.jsonnet")
	utils.EvalJsonnetFrom(config_path, &config)
	slog.Debug("Load Config", "path", config_path, "rendered", config)

	identities_path := utils.InterpretTildeHome(current_user.HomeDir, config.AgeIdentities)
	identities_file, err := os.Open(identities_path)
	if err != nil {
		log.Panicf("Failed to open age identities file `%s`: %s", config.AgeIdentities, err)
	}
	identities, err := age.ParseIdentities(identities_file)
	if err != nil {
		log.Fatalf("Failed to parse age identities file: %s", err)
	}
	slog.Debug("Load Age Identities", "path", identities_path, "identities", identities)

	recipients_path := filepath.Join(working_dir, ".age-recipients")
	recipients_file, err := os.Open(recipients_path)
	if err != nil {
		log.Panicf("Failed to open age keys file `%s`: %s", filepath.Join(working_dir, ".age-recipients"), err)
	}
	recipients, err := age.ParseRecipients(recipients_file)
	if err != nil {
		log.Fatalf("Failed to parse age recipients file: %s", err)
	}
	slog.Debug("Load Age Recipients", "path", recipients_path, "recipients", recipients)

	state_path := filepath.Join(working_dir, "state.json")
	var state st.State
	_, err = os.Stat(state_path)
	if errors.Is(err, os.ErrNotExist) {
		state = st.NewState()
		utils.SaveJsonTo(state_path, &state)
	} else {
		utils.LoadJsonFrom(state_path, &state)
	}
	slog.Debug("Load State", "path", state_path, "content", state)

	hosts := make(map[string]ho.Host)
	for host_name, host_config := range config.Hosts {
		host_state, ok := state.Hosts[host_name]
		if ok {
			hosts[host_name] = ho.Host{Name: host_name, Config: &host_config, State: &host_state}
		} else {
			hosts[host_name] = ho.Host{Name: host_name, Config: &host_config, State: nil}
		}
	}

	return Context{
		WorkingDir:  working_dir,
		CurrentUser: current_user,
		Config:      config,
		Identities:  identities,
		Recipients:  recipients,
		State:       state,
	}
}

func (c *Context) Encrypt(data []byte) string {
	out := &bytes.Buffer{}
	writer, err := age.Encrypt(out, c.Recipients...)
	if err != nil {
		log.Fatal(err)
	}
	writer.Write(data)
	writer.Close()
	return base64.URLEncoding.EncodeToString(out.Bytes())
}

func (c *Context) SaveState() {
	utils.SaveJsonTo(c.StatePath, c.State)
}

func (c *Context) GetSSHPrivateKey(host ho.Host) ssh.Signer {
	decoded_priv_key, err := base64.URLEncoding.DecodeString(host.State.PrivateKey)
	if err != nil {
		log.Panicf("Failed to decode base64 encoded PrivateKey: %s", err)
	}

	slog.Debug("Decoded Base64 Encoded PrivateKey", "host_name", host.Name)

	priv_key_reader := bytes.NewReader(decoded_priv_key)
	priv_key_age_reader, err := age.Decrypt(priv_key_reader, c.Identities...)
	if err != nil {
		log.Panicf("Failed to decrypt Age encrypted PrivateKey: %s", err)
	}
	priv_key, err := io.ReadAll(priv_key_age_reader)
	if err != nil {
		log.Panicf("Failed to decrypt Age encrypted PrivateKey: %s", err)
	}

	slog.Debug("Decrypted Age Encrypted PrivateKey", "priv_key", priv_key)

	signer, err := ssh.ParsePrivateKey(priv_key)
	if err != nil {
		log.Panicf("Failed to parse private key: %s", err)
	}
	return signer
}
