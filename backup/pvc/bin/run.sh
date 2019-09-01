#!/usr/bin/env bash

set -eo pipefail

[[ -z "${BACKUP_DIR}" ]] && echo "Required 'BACKUP_DIR' env not set" && exit 1;
[[ -z "${JENKINS_HOME}" ]] && echo "Required 'JENKINS_HOME' env not set" && exit 1;

while true;
do
    sleep 10
    if [[ ! -z "${BACKUP_COUNT}" ]]; then
        echo "Trimming to only ${BACKUP_COUNT} recent backups in preparation for new backup"
        find ${BACKUP_DIR} -name '*.tar.gz' -exec basename {} \; | sort -gr | tail -n +$((BACKUP_COUNT +1)) | xargs -I '{}' rm ${BACKUP_DIR}/'{}'
    fi
done
