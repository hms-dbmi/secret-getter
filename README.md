# secret-getter

[secret-getter](secret-getter) replaces file and/or environment variables with retrieved Vault or file secrets/key-value pairs.

secret-getter is used with hms-dbmi [Docker Images](https://github.com/hms-dbmi/docker-images/secret-getter) and [Docker Deployments](https://github.com/hms-dbmi/docker-images/tree/master/deployments).

## Vault command line arguments

Available Vault environment variables:

-   VAULT_ADDR
-   VAULT_TOKEN

Command line arguments override environment variables. Prefixes and suffixes are expected to be in regex.

```bash
$ secret_getter vault

    -addr        Vault address
    -token       Vault token or file path to token
    -path        Vault path
    -prefixes    regex prefix
    -suffixes    regex suffix
    -files       List of files to replace with secrets
    -order       Order of precedence: override, vault, or env
```

## File command line arguments

```bash
$ secret_getter file

    -path        path to key/value pair secrets file
    -prefixes    regex prefix
    -suffixes    regex suffix
    -files       List of files to replace with secrets
    -order       Order of precedence: override, vault, or env
```

`secret_getter file` parses `-path=/path/to/file` as key=value pairs

```bash
# example
secret1 = secret_value_1
secret2 = secret_value_2
secret3 = "secret_value 3"
secret4 =
secret5 = ""
```

### Order options:

```bash
    # default option. uses secrets from Vault or file.
    # If value does not exist, environment value is used
    -order=vault
        vault or file value > environment value

    # *** development only ***
    # uses values from environment.
    # If value does not exist, vault/file value is used
    # useful to override read-only secrets
    -order=env
        -environment value > vault or file value

    # *** development only ***
    # uses secrets from Vault or file.
    # Replaces environment values with vault/file values
    # useful for debugging
    -order=override
        both environment and file variables overwritten with vault or file values
```

### Examples

```bash
$ export VAULT_ADDR=https://locahost
$ export VAULT_TOKEN=000-000-0000
$ export VAULT_KEY_1=VALUE_2

# This will replace keys, matching regex ${key},
# found in /path/to/file1 and /path/to/file2
# with values from Vault or the environment
# with environment values having order of precedence
$ secret_getter vault -path=/path/in/Vault/ \
    -files=/path/to/file1,/path/to/file2 \
    -prefix=\$\{ \
    -suffix=\}
    -order=env
```
