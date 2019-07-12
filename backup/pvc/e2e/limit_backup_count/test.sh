#!/bin/bash
set -eo pipefail

[[ "${DEBUG}" ]] && set -x

# set current working directory to the directory of the script
cd "$(dirname "$0")"

docker_image=$1

if ! docker inspect ${docker_image} &> /dev/null; then
    echo "Image '${docker_image}' does not exists"
    false
fi

JENKINS_HOME="$(pwd)/jenkins_home"
BACKUP_DIR="$(pwd)/backup"
mkdir -p ${BACKUP_DIR}

touch ${BACKUP_DIR}/1.tar.gz
touch ${BACKUP_DIR}/2.tar.gz
touch ${BACKUP_DIR}/3.tar.gz
touch ${BACKUP_DIR}/4.tar.gz
touch ${BACKUP_DIR}/5.tar.gz
touch ${BACKUP_DIR}/6.tar.gz
touch ${BACKUP_DIR}/7.tar.gz
touch ${BACKUP_DIR}/8.tar.gz
touch ${BACKUP_DIR}/9.tar.gz
touch ${BACKUP_DIR}/10.tar.gz
touch ${BACKUP_DIR}/11.tar.gz

# Create an instance of the container under testing
cid="$(docker run -e BACKUP_COUNT=2 -e JENKINS_HOME=${JENKINS_HOME} -v ${JENKINS_HOME}:${JENKINS_HOME}:ro -e BACKUP_DIR=${BACKUP_DIR} -v ${BACKUP_DIR}:${BACKUP_DIR}:rw -d ${docker_image})"
echo "Docker container ID '${cid}'"

# Remove test directory and container afterwards
trap "docker rm -vf $cid > /dev/null;rm -rf ${BACKUP_DIR}" EXIT

sleep 11
touch ${BACKUP_DIR}/12.tar.gz
sleep 11

if [[ "${DEBUG}" ]]; then
    docker logs ${cid}
    ls -la ${BACKUP_DIR}
fi

# only two latest backup should exists
[[ $(ls -1 ${BACKUP_DIR} | wc -l) -eq 2 ]] || exit 1
[[ -f ${BACKUP_DIR}/11.tar.gz ]] || exit 2
[[ -f ${BACKUP_DIR}/12.tar.gz ]] || exit 3
echo PASS
