#!/usr/bin/env bash

set -Eeuo pipefail

readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly NC='\033[0m'

error_handler() {
    local exit_code=$?
    local line_number=$1

    echo -e "${RED}Docker installation failed at line ${line_number}.${NC}"
    exit "${exit_code}"
}

trap 'error_handler ${LINENO}' ERR

if [[ "${EUID}" -eq 0 ]]; then
    echo -e "${RED}Please run this script as a normal sudo user, not as root.${NC}"
    exit 1
fi

if ! command -v apt-get >/dev/null 2>&1; then
    echo -e "${RED}This script supports Ubuntu/Debian systems only.${NC}"
    exit 1
fi

echo -e "${YELLOW}Updating system packages...${NC}"
sudo apt-get update -y

echo -e "${YELLOW}Installing required packages...${NC}"
sudo apt-get install -y \
    ca-certificates \
    curl \
    gnupg

echo -e "${YELLOW}Removing conflicting Docker packages, if present...${NC}"
for package in docker.io docker-doc docker-compose podman-docker containerd runc; do
    sudo apt-get remove -y "${package}" 2>/dev/null || true
done

echo -e "${YELLOW}Adding Docker official repository...${NC}"

sudo install -m 0755 -d /etc/apt/keyrings

sudo curl -fsSL \
    https://download.docker.com/linux/ubuntu/gpg \
    -o /etc/apt/keyrings/docker.asc

sudo chmod a+r /etc/apt/keyrings/docker.asc

. /etc/os-release

echo \
    "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu ${VERSION_CODENAME} stable" \
    | sudo tee /etc/apt/sources.list.d/docker.list >/dev/null

sudo apt-get update -y

echo -e "${YELLOW}Installing Docker Engine and Docker Compose...${NC}"

sudo apt-get install -y \
    docker-ce \
    docker-ce-cli \
    containerd.io \
    docker-buildx-plugin \
    docker-compose-plugin

echo -e "${YELLOW}Starting Docker service...${NC}"

sudo systemctl enable --now docker

echo -e "${YELLOW}Adding ${USER} to the docker group...${NC}"

sudo usermod -aG docker "${USER}"

echo -e "${YELLOW}Checking Docker installation...${NC}"

sudo docker version
sudo docker compose version

echo
echo -e "${GREEN}Docker installation completed successfully.${NC}"
echo
echo "Run the following command to apply Docker group permission:"
echo
echo "    newgrp docker"
echo
echo "After that, test Docker using:"
echo
echo "    docker run --rm hello-world"