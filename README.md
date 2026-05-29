# executor

`executor` is a minimalistic alternative to Ansible. It is designed to run from
an administrator workstation, connect to servers over SSH, copy a small
statically linked agent, and then drive that agent through the same SSH
connection to deploy configuration.

The project favors a compact control plane over a large runtime dependency on
managed hosts. The local `executor` binary embeds the remote `agent` binary and
uses SSH as the only transport.

The tool keeps per-host bootstrap state in `state.json`. Private SSH keys in
that state are encrypted with age before they are written to disk.

## Repository Layout

```text
cmd/agent/       Minimal remote agent binary entry point.
cmd/executor/    Administrator workstation CLI entry point.
pkg/config/      Config schema.
pkg/context/     Runtime context loading, age encryption, and state handling.
pkg/host/        Host bootstrap logic.
pkg/state/       State schema.
pkg/utils/       JSON, Jsonnet, SSH, path, and logging helpers.
examples/base/   Minimal example configuration and state shape.
```

## Requirements

- Go matching the version in `go.mod`.
- `upx` for the `agent` build target.
- age identity and recipient files.
- SSH access as `root` to each target server during bootstrap.

The current bootstrap flow initially connects with password authentication using
the password `root`, installs a generated SSH public key, copies the embedded
agent to `~/agent`, and then stores the generated private key in encrypted
state.

## Build

```sh
make
```

This builds both binaries:

- `agent`
- `executor`

The `agent` target also compresses the agent with `upx` and copies it to
`cmd/executor/embed/agent` so `executor` can embed it.

To clean generated build artifacts:

```sh
make clean
```

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
