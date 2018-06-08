/*

$ secret-getter vault

available environment variables:
- VAULT_ADDR
- VAULT_TOKEN

command line arguments:
    addr     = flag.String("addr", "", "Vault address")
    token    = flag.String("token", "", "Vault token")
    path     = flag.String("path", "", "Vault path")
    prefixes = flag.String("prefix", "{", "Front prefix")
    suffixes = flag.String("suffix", "}", "End prefix")
    files    = flag.String("files", "", "List of files to replace with Vault secrets")
    order    = flag.String("order", "vault", "Order of precedence: override, vault, or env")

command line arguments override environment variables.

prefixes and suffixes are expected to be in regex.

order options:
    vault preference (vault)
        - vault key > environment key
    env preference (env)
        - environment key > vault key
    vault override (override)
        - environment variables assigned vault values

e.g.
VAULT_ADDR=https://locahost
VAULT_TOKEN=000-000-0000
VAULT_KEY_1=VALUE_2

secret_getter vault -path=/path/in/Vault/ -files=/path/to/file1,/path/to/file2 -prefix=\$\{ -suffix=\} -order=env

This will replace keys, matching regex ${key}, found in /path/to/file1 and /path/to/file2
with environment or Vault values, enviroment values having order of precedence

*/

package main
