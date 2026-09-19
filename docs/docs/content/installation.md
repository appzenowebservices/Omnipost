# Installation

patra is a simple binary application that requires a Postgres database instance to run. The binary can be downloaded and run manually, or it can be run as a container with Docker compose.

## Binary
1. Download the [latest release](https://github.com/appzenowebservices/Omnipost/releases) and extract the patra binary. `amd64` is the main one. It works for Intel and x86 CPUs.
1. `./patra --new-config` to generate config.toml. Edit the file.
1. `./patra --install` to install the tables in the Postgres DB (⩾ 12).
1. Run `./patra` and visit `http://localhost:9000` to create the Super Admin user and login.

!!! Tip
    To set the Super Admin username and password during installation, set the environment variables:
    `PATRA_ADMIN_USER=myuser PATRA_ADMIN_PASSWORD=xxxxx ./patra --install`


## Docker

The latest image is available on DockerHub at `patra/patra:latest`

The recommended method is to download the [docker-compose.yml](https://github.com/appzenowebservices/Omnipost/blob/master/docker-compose.yml) file, customize it for your environment and then to simply run `docker compose up -d`.

```shell
# Download the compose file to the current directory.
curl -LO https://github.com/appzenowebservices/Omnipost/raw/master/docker-compose.yml

# Run the services in the background.
docker compose up -d
```

Then, visit `http://localhost:9000` to create the Super Admin user and login.

!!! Tip
    To set the Super Admin username and password during setup, set the environment variables (only the first time):
    `PATRA_ADMIN_USER=myuser PATRA_ADMIN_PASSWORD=xxxxx docker compose up -d`


### Mounting a custom config.toml
The docker-compose file includes all necessary patra configuration as environment variables, `PATRA_*`.
If you would like to remove those and mount a config.toml instead:

#### 1. Save the config.toml file on the host

```toml
[app]
address = "0.0.0.0:9000"

# Database.
[db]
host = "patra_db" # Postgres container name in the compose file.
port = 5432
user = "patra"
password = "patra"
database = "patra"
ssl_mode = "disable"
max_open = 25
max_idle = 25
max_lifetime = "300s"
```

#### 2. Mount the config file in docker-compose.yml

```yaml
  app:
    ...
    volumes:
    - /path/on/your/host/config.toml:/patra/config.toml
```

#### 3. Change the `--config ''` flags in the `command:` section to point to the path

```yaml
command: [sh, -c, "./patra --install --idempotent --yes --config /patra/config.toml && ./patra --upgrade --yes --config /patra/config.toml && ./patra --config /patra/config.toml"]
```

-----------

## Nightly

!!! Warning
    Nightly releases are untested and may have bugs. Use at your own risk. Always take a backup of your Postgres database before using a nightly release.

A nightly build is automatically published with the latest changes merged to the repository. If you want to access the latest changes without waiting for versioned releases, you can obtain the nightly builds and follow the same instructions above.

- **Docker**: `patra/patra:nightly` (use this as the image name in the docker-compose file)
- **Binary**: [Download nightly release](https://github.com/appzenowebservices/Omnipost/releases/nightly)


## Compiling from source

To compile the latest unreleased version (`master` branch):

1. Make sure `go`, `nodejs`, and `yarn` are installed on your system.
2. `git clone git@github.com:knadh/patra.git`
3. `cd patra && make dist`. This will generate the `patra` binary.


## Helm chart for Kubernetes

A Helm chart for installing patra on a Kubernetes cluster is maintained by the community in the [HelmForge charts repository](https://github.com/helmforgedev/charts/tree/main/charts/patra).

To use the Helm chart, configure `values.yaml` according to your needs, and then run the following command:

```shell
$ helm upgrade \
    --create-namespace \
    --install patra patra \
    --namespace patra \
    --repo https://repo.helmforge.dev \
    --values values.yaml
```

## 3rd party hosting

<a href="https://dash.elest.io/deploy?soft=Patra&id=237"><img src="https://raw.githubusercontent.com/elestio-examples/reactjs/refs/heads/master/src/deploy-on-elestio.png" alt="Deploy to Elestio" height="35" style="max-width: 150px;" /></a>
<br />
<a href="https://www.pikapods.com/pods?run=patra"><img src="https://www.pikapods.com/static/run-button.svg" alt="Deploy on PikaPod" style="max-width: 150px;" /></a>
<br />
<a href="https://northflank.com/stacks/deploy-patra"><img src="https://assets.northflank.com/deploy_to_northflank_smm_36700fb050.svg" alt="One-click deploy on Northflank" height="35" style="max-width: 150px; border-radius: 6px; object-fit: contain;" /></a>
<br />
<a href="https://railway.app/new/template/patra"><img src="https://railway.app/button.svg" alt="One-click deploy on Railway" style="max-width: 150px;" /></a>
<br />
<a href="https://repocloud.io/details/?app_id=217"><img src="https://d16t0pc4846x52.cloudfront.net/deploy.png" alt="Deploy at RepoCloud" style="max-width: 150px;"/></a>
<br />
<a href="https://sealos.io/products/app-store/patra"><img src="https://sealos.io/Deploy-on-Sealos.svg" alt="Deploy on Sealos" style="max-width: 150px;"/></a>
<br />
<a href="https://zeabur.com/templates/5EDMN6"><img src="https://zeabur.com/button.svg" alt="Deploy on Zeabur" style="max-width: 150px;"/></a>
<br />
<a href="https://www.cloudron.io/store/app.patra.cloudronapp.html"><img src="https://cloudron.io/img/button.svg" alt="Install on Cloudron" style="max-width: 150px;"/></a>
<br />
<a href="https://console.hostim.dev/dashboard?preview=1&modal=1&template=patra"><img src="https://hostim.dev/img/deploy-button.svg" alt="Deploy on Hostim" style="max-width: 150px;"/></a>

## Tutorials
* [Patra with Forward Email for Secure Newsletter Delivery](https://forwardemail.net/en/guides/newsletter-with-patra)
* [Informal step-by-step on how to get started with patra using *Railway*](https://github.com/appzenowebservices/Omnipost/issues/120#issuecomment-1421838533)
* [Step-by-step tutorial for installation and all basic functions. *Amazon EC2, SES, docker & binary*](https://gist.github.com/MaximilianKohler/e5158fcfe6de80a9069926a67afcae11)
* [Step-by-step guide on how to install and set up patra on *AWS Lightsail with docker* (rameerez)](https://github.com/appzenowebservices/Omnipost/issues/1208)
* [Quick setup on any cloud server using *docker and caddy*](https://github.com/samyogdhital/patra-caddy-reverse-proxy)
* [*Binary* install on Ubuntu 22.04 as a service](https://mumaritc.hashnode.dev/how-to-install-patra-using-binary-on-ubuntu-2204)
* [*Binary* install on Ubuntu 18.04 as a service (Apache & Plesk)](https://devgypsy.com/post/2020-08-18-installing-patra-newsletter-manager/)
* [*Binary and docker* on linux (techviewleo)](https://techviewleo.com/manage-mailing-list-and-newsletter-using-patra/)
* [*Binary* install on your PC](https://www.youtube.com/watch?v=fAOBqgR9Yfo). Discussions of limitations: [[1](https://github.com/appzenowebservices/Omnipost/issues/862#issuecomment-1307328228)][[2](https://github.com/appzenowebservices/Omnipost/issues/248#issuecomment-1320806990)].
* [*Docker on Rocky Linux 8* (nginx, Let's Encrypt SSL)](https://wiki.crowncloud.net/?How_to_Install_Patra_with_Docker_on_Rocky_Linux_8)
* [*Docker* with nginx reverse proxy, certbot SSL, and Gmail SMTP](https://www.maketecheasier.com/create-own-newsletter-with-patra/)
* [Install Patra on Self-hosting with *Pre-Configured AMI Package at AWS* by Single Click](https://meetrix.io/articles/how-to-install-llama-2-on-aws-with-pre-configured-ami-package/)
* [*Fly.io* working example](https://gitlab.com/votelog/apps/newsletter)
