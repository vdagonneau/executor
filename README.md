# executor

`executor` is a minimalistic alternative to Ansible. It is designed to operate on
a push "agentless" model.

You run the main CLI tool from a control node, it connects through SSH to remote
nodes, copies a minimal agent and then applies the locally defined configuration
to the remote node. 

`executor` favors a compact tool over a large runtime dependency on managed hosts.
The local `executor` binary embeds the remote `agent` binary and uses SSH as the
only transport.

`executor` aalso keeps per-host state in `state.json`. Private SSH keys in that
state are encrypted with [age](https://pkg.go.dev/filippo.io/age) before they are
written to disk.

## Configure

Create these files in the working directory where you run `executor`:

- `config.jsonnet`
- `.age-recipients`
- `state.json` if you want to seed existing state; otherwise it is created
  automatically.

`config.jsonnet` defines the age identity file and target hosts:

```jsonnet
{
  age_identities: '~/.config/age/identities.txt',
  hosts: {
    workstation: {
      hostname: '127.0.0.1',
      port: 22,
    },
  },
}
```

`.age-recipients` must contain one or more age recipients that can decrypt with
the configured identities.

See [docs/configuration.md](docs/configuration.md) for the full file contract.

## Run

Build first, then run from a directory containing the configuration files:

```sh
./executor
```

For more verbose diagnostic logs:

```sh
LOG_LEVEL=debug ./executor
```

## Operating Model

`executor` runs locally from an administrator workstation. Target servers do not
need a preinstalled Python runtime or a long-running daemon. During bootstrap,
`executor` installs its own minimal agent on the server and records the SSH
state needed to reconnect later.

After bootstrap, the intended deployment path is:

1. Connect to the server over SSH.
2. Verify or refresh the remote agent.
3. Use the agent through the SSH session to apply configuration to the server.

The current implementation bootstraps hosts and verifies the remote agent
version. Configuration deployment is the next layer on top of the established
SSH and agent transport.

## What Happens On Each Run

For each configured host:

1. If the host is missing from `state.json`, `executor` bootstraps it.
2. During bootstrap, `executor` records the host key, generates a new SSH key
   pair, installs the public key, installs the agent, encrypts the private key,
   and saves state.
3. If the host already has state, `executor` connects with the saved key and a
   fixed host key check.
4. It runs `./agent --version` remotely and warns if the remote agent version
   differs from the local embedded build version.

## Security Notes

- `state.json` includes host keys and encrypted private keys. Treat it as
  sensitive operational state.
- The first bootstrap trust decision accepts the host key seen during the
  initial SSH connection and stores it for future fixed-key verification.
- The initial bootstrap path currently assumes root password authentication.

## More Documentation

- [Configuration](docs/configuration.md)
- [Architecture](docs/architecture.md)
