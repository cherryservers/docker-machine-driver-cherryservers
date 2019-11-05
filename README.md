# Cherry Servers Cloud Docker machine driver

[![License](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

> This library adds the support for creating [Docker machines](https://github.com/docker/machine) hosted on the [Cherry Servers](https://www.cherryservers.com/).

You need to create auth token under `API Keys` > `Create API key` in the client control panel
and pass that to `docker-machine create` with the `--cherryservers-auth-token` option.

## Installation

You can find pre-compiled binaries [here](http://downloads.cherryservers.com/other/docker-machine-driver/).

You need to download appropriate driver to your PATH location, for example:


```bash
# MAC
wget http://downloads.cherryservers.com/other/docker-machine-driver/mac/docker-machine-driver-cherryservers \
    -O /usr/local/bin/docker-machine-driver-cherryservers

# LINUX
wget http://downloads.cherryservers.com/other/docker-machine-driver/linux/docker-machine-driver-cherryservers \
    -O /usr/local/bin/docker-machine-driver-cherryservers
chmod +x /usr/local/bin/docker-machine-driver-cherryservers
```


## Usage

```bash
$ docker-machine create -d cherryservers \
	--cherryservers-hostname "hostname.host.com" \
	--cherryservers-project-id "79813" \
	--cherryservers-auth-token "bnRfaWQiOjQ5Nzk3LCJpYXQiOjE1NjM1MjMyNzR9.iUCq4JxHYjXu"  \
	--cherryservers-ssh-key-label "95" \
--cherryservers-ssh-key-path "/path/to/ssh/key/id_rsa" machine_name
```

In this case your public SSH key will be uploaded to client portal and added to new deployed servers so docker-machine can access it.

### Using existing key in client portal

```bash
$ docker-machine create -d cherryservers \
	--cherryservers-hostname "hostname.host.com" \
	--cherryservers-project-id "79813" \
	--cherryservers-auth-token "bnRfaWQiOjQ5Nzk3LCJpYXQiOjE1NjM1MjMyNzR9.iUCq4JxHYjXu" \
	--cherryservers-existing-ssh-key-path "/path/to/ssh/key/id_rsa" \
	--cherryservers-existing-ssh-key-label "key_label_in_portal" \
machine_name
```

In that case you public key won't be uploaded to portal but existing key will be used. Private key's fingerprint must match local key's fingerprint in that case.

### Generating new key pair

```bash
$ docker-machine create -d cherryservers \
	--cherryservers-hostname "hostname.host.com" \
	--cherryservers-project-id "79813" \
	--cherryservers-auth-token "bnRfaWQiOjQ5Nzk3LCJpYXQiOjE1NjM1MjMyNzR9.iUCq4JxHYjXu" \
machine_name
```

In that case new SSH keypair will be generated ant new public key will be uploaded to client portal with name of machine name.

## Options

- `--cherryservers-auth-token`: **required**. Your auth token for the Cherry Servers API.
- `--cherryservers-project-id`: **required**. Your project ID.
- `--cherryservers-hostname`: Your defined server hostname.
- `--cherryservers-existing-ssh-key-path`: Path to your ssh key's private key.
- `--cherryservers-existing-ssh-key-label`: Label of existing public SSH key in client portal.
- `--cherryservers-region`: server region ("EU-East-1" or "EU-West-1").
- `--cherryservers-image`: your server image e.g. `Ubuntu 16.04 64bit`.
- `--cherryservers-plan`: your server plan ID.


#### Environment variables and default values

| CLI option                                | Environment variable                      | Default                    |
| ----------------------------------------- | ----------------------------------------- | -------------------------- |
| **`--cherryservers-auth-token`**          | `CHERRYSERVERS_AUTH_TOKEN`                | -                          |
| `--cherryservers-project-id`              | `CHERRYSERVERS_PROJECT_ID`                | -                          |
| `--cherryservers-hostname`                | `CHERRYSERVERS_HOSTNAME`                  | -                          |
| `--cherryservers-existing-ssh-key-path`   | `CHERRYSERVERS_EXISTING_SSH_KEY_PATH`     | -                          |
| `--cherryservers-existing-ssh-key-label`  | `CHERRYSERVERS_EXISTING_SSH_KEY_LABEL`    | -                          |
| `--cherryservers-region`                  | `CHERRYSERVERS_REGION`                    | `EU-East-1`                |
| `--cherryservers-image`                   | `CHERRYSERVERS_IMAGE`                     | `Ubuntu 16.04 64bit`       |
| `--cherryservers-plan`                    | `CHERRYSERVERS_PLAN`                      | `94`                       |
