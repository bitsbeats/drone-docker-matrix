# drone-docker-matrix

This is a Drone plugin to build a lot of Docker images.

## Usage

### Plugin configuration

- `PLUGIN_REGISTRY`: Registry to upload the image to. (*required*)
- `PLUGIN_DEFAULT_NAMESPACE`: Namespace to use if not specified in `docker-matrix.yml` (default: `images`).
- `PLUGIN_BUILD_POOL_SIZE`: Number of parallel Docker builds (default: `4`).
- `PLUGIN_UPLOAD_POOL_SIZE`: Number of parallel Docker uploads (default: `4`).
- `PLUGIN_TAG_NAME`: Tag Name (default: `latest`).

**NOTE**: For values in `PLUGIN_TAG_NAME` one may choose to use environment variables. Substition is handled by [drone/envsubst](https://github.com/drone/envsubst)

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
* `append` options are just added as they are (*optional*).
* `namespace` can overwrite the `DEFAULT_NAMESPACE` variable (*optional*).
* `additional_names` can supply additional image-names to upload to, i.e. to other registries (*optional*).

**NOTE**: For values in `multiply`, `append`, and `namespace` one may choose to use environment variables. Substition is handled by [drone/envsubst](https://github.com/drone/envsubst)


```yaml
# docker-matrix.yml
multiply:
  VERSION:
    - 7.2
    - 7.3
  OS:
    - alpine
    - debian
    
append:
  - { NAME: test, LANG: ${LANG} }
  
namespace: images

additional_names:
  - docker.io/bitsbeats/image1
  - docker.io/bitsbeats/image2
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

### Running without Drone

Example:

```bash
export PLUGIN_WORKDIR=/home/user42/docker-images
export PLUGIN_REGISTRY=registry.example.com
export PLUGIN_DEFAULT_NAMESPACE=images

drone-docker-matrix
```
