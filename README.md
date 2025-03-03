# drone-docker-matrix

This is a Drone plugin to build a lot of Docker images.

## Usage

### Plugin configuration

- `PLUGIN_REGISTRY`: Registry to upload the image to. (*required*)
- `PLUGIN_DEFAULT_NAMESPACE`: Namespace to use if not specified in `docker-matrix.yml` (default: `images`).
- `PLUGIN_BUILD_POOL_SIZE`: Number of parallel Docker builds (default: `4`).
- `PLUGIN_UPLOAD_POOL_SIZE`: Number of parallel Docker uploads (default: `4`).
- `PLUGIN_TAG_NAME`: Tag Name (default: `latest`).
- `PLUGIN_TAG_BUILD_ID`: Build id, generates `tag` and `tag-b<build_id>` for each tag; skipped if empty (default *empty*).
- `PLUGIN_SKIP_UPLOAD`: Skip upload to registries, useful for testing (default `false`)
- `PLUGIN_PULL`: Try to pull all docker images (default `true`)

**NOTE**: For values in `PLUGIN_TAG_NAME` and `PLUGIN_TAG_ID` one may choose to use environment variables. Substition is handled by [drone/envsubst](https://github.com/drone/envsubst)

### Repository data

The subdirectories are the image names.

```
puppet
puppet/Dockerfile
puppet/Gemfile

python
python/Dockerfile

php/
php/docker-matrix.yml
php/Dockerfile

[...]
```

The `puppet` and the `python` image are build as they are. The php image will get the *special* matrix treatment.

### Matrixfile

* `multiply` options will get multiplied with each other (*optional*).
* `append` options are just added to all multiplied builds (*optional*).
* `custom_builds`: these are additional builds, you need to specify all options in here (*optional*)
* `namespace` can overwrite the `DEFAULT_NAMESPACE` variable (*optional*).
* `additional_names` can supply additional image-names to upload to, i.e. to other registries (*optional*).
* `as_latest`: image with the supplied tag will be tagged as latest (*optional*).

**NOTE**: For values in `multiply`, `append`, and `namespace` one may choose to use environment variables. Substition is handled by [drone/envsubst](https://github.com/drone/envsubst)

The `multiply` can have an empty string as field. This wont be added to the images tag. Useful for default options. You can use Bash Syntax to use a default value instead: `echo ${MESSAGE}:-default`.

```yaml
# docker-matrix.yml
multiply:
  VERSION:
    - 7.2
    - 7.3
  OS:
    - alpine
    - debian
  COMMAND:
    - sleep 1y
    - ""

append:
  - { NAME: test, LANG: ${LANG} }

namespace: images

additional_names:
  - docker.io/bitsbeats/image1
  - docker.io/bitsbeats/image2

as_latest: 7.2-debian
```

The Dockerfile has to contain the argument names and default values. Example:

```Dockerfile
ARG \
  VERSION=7.3 \
  OS=debian \
  NAME=test

FROM php:$VERSION-fpm-$OS

RUN touch $NAME
```

### Building external repositories

It's possible to build Dockerfiles from an external repository. The path to the
repository is specified just like for the `docker build` cli command. The syntax
is described [here](https://docs.docker.com/engine/reference/commandline/build/#git-repositories).

* `custom_path`: use a specific path to build (*optional*)
* `custom_dockerfile`: use a specific dockerfile (*optional*, default: `Dockerfile`)

```yaml
# docker-matrix.yml
custom_path: https://github.com/openshift/origin-aggregated-logging.git#release-3.11:fluentd
custom_dockerfile: Dockerfile.centos7
```

**Note**: Currently its still required that a Dockerfile exists next to the
`drone-matrix.yml`, since we use Dockerfiles to discover. The file can be empty
or just a comment.

### Running without Drone

Example:

```bash
export PLUGIN_WORKDIR=/home/user42/docker-images
export PLUGIN_REGISTRY=registry.example.com
export PLUGIN_DEFAULT_NAMESPACE=images

drone-docker-matrix
```
