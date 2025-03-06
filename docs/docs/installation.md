# Installation

Libredesk is a single binary application that requires postgres and redis to run. You can install it using the binary or docker.

## Binary

1. Download the [latest release](https://github.com/abhinavxd/libredesk/releases) and extract the libredesk binary.
2. `./libredesk --install` to install the tables in the Postgres DB (⩾ 13) and set the System user password.
3. Run `./libredesk` and visit `http://localhost:9000` and login with the email `System` and the password you set during installation.

!!! Tip
    To set the System user password during installation, set the environment variables:
    `LIBREDESK_SYSTEM_USER_PASSWORD=xxxxxxxxxxx ./libredesk --install`


## Docker

The latest image is available on DockerHub at `libredesk/libredesk:latest`

The recommended method is to download the [docker-compose.yml](https://github.com/abhinavxd/libredesk/blob/main/docker-compose.yml) file, customize it for your environment and then to simply run `docker compose up -d`.

```shell
# Download the compose file and the sample config file in the current directory.
curl -LO https://github.com/abhinavxd/libredesk/raw/main/docker-compose.yml
curl -LO https://github.com/abhinavxd/libredesk/raw/main/config.sample.toml

# Copy the config.sample.toml to config.toml and edit it as needed.
cp config.sample.toml config.toml

# Run the services in the background.
docker compose up -d

# Setting System user password.
docker exec -it libredesk_app ./libredesk --set-system-user-password
```

Go to `http://localhost:9000` and login with the email `System` and the password you set using the `--set-system-user-password` command.


## Compiling from source

To compile the latest unreleased version (`main` branch):

1. Make sure `go`, `nodejs`, and `pnpm` are installed on your system.
2. `git clone git@github.com:abhinavxd/libredesk.git`
3. `cd libredesk && make`. This will generate the `libredesk` binary.


## Nginx

Libredesk using websockets for real-time updates. If you are using Nginx, you need to add the following (or similar) configuration to your Nginx configuration file.

```nginx
location / {
    proxy_pass http://localhost:9000;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection 'upgrade';
    proxy_set_header Host $host;
    proxy_cache_bypass $http_upgrade;
}
```