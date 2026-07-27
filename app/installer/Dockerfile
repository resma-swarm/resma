# syntax=docker/dockerfile:1.7
#
# RESMA Installer — Container installer (padrão SwarmPit).
#
# Roda isolado num container com Docker CLI + docker.sock montado do host.
# Gera secrets, faz pull das imagens, deploya o stack e aguarda healthcheck.
#
# Uso:
#   docker run -it --rm \
#     --name resma-installer \
#     --volume /var/run/docker.sock:/var/run/docker.sock \
#     resmaswarm/resma-install:latest
#
# Modo não-interativo (env vars):
#   docker run -it --rm \
#     --name resma-installer \
#     --volume /var/run/docker.sock:/var/run/docker.sock \
#     -e INTERACTIVE=0 \
#     -e STACK_NAME=resma \
#     -e APP_PORT=8080 \
#     resmaswarm/resma-install:latest
#
# Build:
#   docker build -t resmaswarm/resma-install:latest -f app/installer/Dockerfile .

FROM docker:27-cli

RUN apk add --no-cache curl jq openssl bash

WORKDIR /install

# docker-stack.yml baked na imagem — sem git clone, sem dependência de internet
# para o stack file (só para pull das imagens de runtime).
COPY docker-stack.yml ./docker-stack.yml
COPY app/installer/install.sh ./install.sh
COPY app/installer/logo.txt ./logo.txt

RUN chmod +x install.sh

CMD ["bash", "/install/install.sh"]
