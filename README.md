# concourse-vault-secrets-resource

Concourse resource for retrieving secrets from Hashicorps' Vault.
This proably WILL leave your secrets lingering on disk with running workers, so make sure your workers are secure and trusted...

This is a work in progress. Use at your own risk.

## Source Configuration

```json
{
  "version": "some-version-identifier",
  "source": {
    "vault_uri": "...",
    "app_id": "..."
  }
}
```


### Example

## Behavior

### `check`:

Returns a checksum of the expiration time of the current lease as a version.

### `in`:

Retrieve a secrets named in the source config (values) and generates a yaml file with the keys from the config mapping to the actual secrets.

Get parameters:

```json
...
"params": {
  "secrets": {
    "secret-name": "seceret-path",
    ....
  }
},
...
```

### `out`:

Not supported for now; We consider Vault as readonly for concourse.
