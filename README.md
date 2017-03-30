# vault-getter
retrieves secrets given a vault_token, and replaces values in files and/or environment variables

vailable environment variables:

- VAULT_ADDR
- VAULT_PATH
- VAULT_TOKEN

command line arguments:

    --addr        Vault address
    --token       Vault token
    --path        Vault path
    --prefixes    regex prefix
    --suffixes    regex suffix
    --files       List of files to replace with Vault secrets
    --order       Order of precedence: override, vault, or env

command line arguments override environment variables.
prefixes and suffixes are expected to be in regex.

order options:

    vault preference (vault)
        - vault key > environment key
        
    env preference (env)
        - environment key > vault key
        
    vault override (override)
        - environment variables and files overwritten with vault values

e.g.:

	VAULT_ADDR=locahost
	VAULT_TOKEN=000-000-0000
	VAULT_KEY_1=VALUE_2
	
	vault_getter --path=/path/in/Vault/ --files=/path/to/file1,/path/to/file2 --prefix=\$\{ --suffix=\} --order=env

This will replace keys, matching regex ${key}, found in /path/to/file1 and /path/to/file2 with values from Vault or the environment, with enviroment values having order of precedence

