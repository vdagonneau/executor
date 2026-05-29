# Architecture

`executor` is organized as a small administrator-workstation control plane plus
a minimal remote agent. The workstation binary owns orchestration, state, and
SSH connections. The server-side agent is copied on demand and is intended to
apply configuration commands sent through the SSH session.

The design is intentionally closer to a small Ansible alternative than to a
daemon-based management system: SSH is the transport, the agent is lightweight,
and host state stays with the administrator workspace.

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

## Binaries

### `cmd/agent`

`agent` is the minimal statically linked binary installed on remote servers. It
is the execution surface that `executor` uses after connecting over SSH.

Its current command surface is:

```sh
./agent version
./agent copy --filename /path/to/file < base64-payload
```

The version value is injected at build time from the current Git commit hash.
The `copy` action reads standard base64 data from stdin, decodes it, and writes
the decoded bytes to the filename passed with `--filename`.

### `cmd/executor`

`executor` is the administrator workstation orchestration binary. It embeds the
built `agent` binary via `go:embed`, loads the runtime context, and iterates
through every configured host.

For hosts without state, it bootstraps the server by installing the agent and
recording connection state. For hosts with state, it checks SSH connectivity and
compares the remote agent version with the embedded local version.

The intended configuration deployment path builds on that connection: once the
agent is present, `executor` can invoke it through SSH to apply server
configuration without requiring a larger managed-host runtime.

## Build Flow

`make agent`:

1. Builds `cmd/agent`.
2. Injects `main.commitHash` with `git rev-parse HEAD`.
3. Builds a static binary.
4. Compresses it with `upx`.
5. Copies it to `cmd/executor/embed/agent`.

`make executor`:

1. Builds `cmd/executor`.
2. Injects `main.commitHash` with `git rev-parse HEAD`.
3. Embeds `cmd/executor/embed/agent`.

## Runtime Flow

1. `context.NewContext` reads the current working directory.
2. It evaluates `config.jsonnet`.
3. It opens and parses the configured age identities file.
4. It opens and parses `.age-recipients`.
5. It loads or creates `state.json`.
6. It joins config and state into a host map.
7. `executor` loops over the host map from the administrator workstation.

Bootstrap path:

1. Connect to the host as `root` using password authentication.
2. Record the host key returned by the SSH server.
3. Generate a new Ed25519 SSH key pair.
4. Append the generated public key to `~/.ssh/authorized_keys`.
5. Copy the embedded statically linked agent to `~/agent` and make it
   executable.
6. Encrypt the generated private key with age recipients.
7. Save host key and encrypted private key in `state.json`.

Existing-state path:

1. Parse the recorded host key.
2. Decrypt and parse the stored private key.
3. Connect as `root` with public key authentication and fixed host key
   verification.
4. Run `./agent version`.
5. Warn if the remote version differs from the local commit hash.
6. Run the configured host actions through the remote agent.

Configuration deployment path:

1. Reuse the established SSH connection to the server.
2. Invoke the remote agent through that SSH connection.
3. For `copy`, send base64-encoded local file bytes to `./agent copy`.
4. Let the agent apply the server-side change locally.

The current code implements bootstrap, version verification, and the `copy`
configuration operation on top of the existing agent-over-SSH shape.

## Package Responsibilities

| Package | Responsibility |
| --- | --- |
| `pkg/config` | Defines the Jsonnet-decoded config schema. |
| `pkg/context` | Builds the runtime context, loads age keys, loads state, encrypts state data, and returns SSH signers. |
| `pkg/host` | Implements server bootstrap, agent installation, and SSH key generation. |
| `pkg/state` | Defines persisted state and empty-state construction. |
| `pkg/utils` | Provides JSON, Jsonnet, logging, path, and SSH command helpers. |

## Operational Boundaries

- `executor` expects all runtime files to be relative to the administrator
  workspace where it is launched.
- `state.json` host entries are keyed by host name from `config.jsonnet`.
- Bootstrap is intentionally one-time per host state entry. Removing a host from
  `state.json` causes bootstrap to run again for that server.
- SSH host key verification is strict after bootstrap because the stored host
  key is reused with `ssh.FixedHostKey`.
- Servers only need SSH access and the copied agent; orchestration state remains
  local to the administrator workstation.
