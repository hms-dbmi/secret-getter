# vault-getter

Replaces file and/or environment variables with retrieved vault secrets

available environment variables:

- VAULT_ADDR
- VAULT_PATH
- VAULT_TOKEN

command line arguments:

```
 --addr        Vault address
 --token       Vault token
 --path        Vault path
 --prefixes    regex prefix
 --suffixes    regex suffix
 --files       List of files to replace with Vault secrets
 --order       Order of precedence: override, vault, or env
```

Command line arguments override environment variables. Prefixes and suffixes are expected to be in regex.

order options:

```
vault preference (vault)
    - vault key > environment key

env preference (env)
    - environment key > vault key

vault override (override)
    - both environment and file variables overwritten with vault values
```

e.g.:

```
VAULT_ADDR=locahost
VAULT_TOKEN=000-000-0000
VAULT_KEY_1=VALUE_2

vault_getter --path=/path/in/Vault/ --files=/path/to/file1,/path/to/file2 --prefix=\$\{ --suffix=\} --order=env
```

This will replace keys, matching regex ${key}, found in /path/to/file1 and /path/to/file2 with values from Vault or the environment, with environment values having order of precedence

_TODO: Abstract vault-getter client out to use any 3rd-party secret repository (secret-getter client)_
