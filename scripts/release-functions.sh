#!/bin/sh

clone_if_not_existent(){
local DEST_DIR=$1
local GIT_REPO=$2

  if [ ! -d $DEST_DIR ]; then
    echo "Cannot find $DEST_DIR: Cloning $GIT_REPO"
    git clone --branch ocp-tools-4.6-rhel-8 $GIT_REPO $DEST_DIR
  fi
}
