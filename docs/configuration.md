# Configuration

`executor` reads configuration and state from the current working directory on
the administrator workstation. The same binary can therefore manage different
server environments depending on the directory where it is launched.

## Required Files

### `config.jsonnet`

`config.jsonnet` is evaluated with Jsonnet and decoded into the Go config
schema.

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

Fields:

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `age_identities` | string | yes | Path to an age identities file. `~` and `~/...` are expanded to the current user's home directory. |
| `hosts` | object | yes | Map of server names to host configuration objects. |
| `hosts.<name>.hostname` | string | yes | Hostname or IP address used for SSH connections. |
| `hosts.<name>.port` | number | yes | SSH port. |

Host map keys are stable server names used in `state.json`.

### `.age-recipients`

`.age-recipients` contains the age recipients used to encrypt generated SSH
private keys before writing them into `state.json`.

Each recipient must be decryptable by one of the identities in
`age_identities`.

### `state.json`

`state.json` stores per-server bootstrap state in the administrator workspace.
If it does not exist, `executor` creates it with an empty host map.

State shape:

```json
{
  "hosts": {
    "workstation": {
      "host_key": "ssh-ed25519 ...",
      "private_key": "base64-url-encoded-age-ciphertext"
    }
  }
}
```

Fields:

| Field | Type | Description |
| --- | --- | --- |
| `hosts.<name>.host_key` | string | SSH server host public key recorded during bootstrap. |
| `hosts.<name>.private_key` | string | Base64 URL encoded age ciphertext containing the generated SSH private key. |

## Log Level

Logging is controlled with `LOG_LEVEL`.

Supported values:

- `debug`
- `info`
- `warn`
- `error`

If `LOG_LEVEL` is unset or unknown, structured logs are effectively silent.

## Example

The `examples/base` directory contains a minimal Jsonnet config and example
state document. Copy the shape rather than the encrypted values; generated keys
and host keys are environment-specific.
