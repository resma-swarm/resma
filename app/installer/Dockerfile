# syntax=docker/dockerfile:1.7
#
# RESMA Installer — Container installer/uninstaller (padrão SwarmPit).
#
# Roda isolado num container com Docker CLI + docker.sock montado do host.
# Gera secrets, faz pull das imagens, deploya o stack e aguarda healthcheck.
#
# Instalar (interativo):
#   docker run -it --rm \
#     --name resma-installer \
#     --volume /var/run/docker.sock:/var/run/docker.sock \
#     resmaswarm/resma-install:latest
#
# Instalar com flags (não-interativo, estilo Docker install):
#   docker run -it --rm \
#     --volume /var/run/docker.sock:/var/run/docker.sock \
#     resmaswarm/resma-install:latest \
#     --no-input \
#     --stack-name resma \
#     --port 8080 \
#     --network captain-overlay-network
#
# Desinstalar:
#   docker run -it --rm \
#     --name resma-uninstaller \
#     --volume /var/run/docker.sock:/var/run/docker.sock \
#     -e MODE=uninstall \
#     resmaswarm/resma-install:latest
#
# Modo não-interativo (env vars — compatível com flags):
#   docker run -it --rm \
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
COPY app/installer/upgrade.sh ./upgrade.sh
COPY app/installer/uninstall.sh ./uninstall.sh
COPY app/installer/entrypoint.sh ./entrypoint.sh
COPY app/installer/logo.txt ./logo.txt

RUN chmod +x install.sh upgrade.sh uninstall.sh entrypoint.sh

ENTRYPOINT ["/install/entrypoint.sh"]
