#!/bin/sh

. $(dirname $(realpath $0))/release-functions.sh

if [[ ! -z $(git status -s) ]] # Uncommitted changes: we exit
then
  echo "There are uncommited changes in local repository. Release preparation cannot continue."
  echo "Commit all changes or run make redhat-release from an up to date branch"
  exit 1
else
  JENKINS_OPERATOR_COMMIT=$(git rev-parse HEAD)
  echo "Current commit is: $JENKINS_OPERATOR_COMMIT"
  NUMBER_OF_TAGS_FOR_COMMIT=$(git tag --points-at $JENKINS_OPERATOR_COMMIT | wc -l)
  TAG_FOR_CURRENT_COMMIT=$(git tag --points-at $JENKINS_OPERATOR_COMMIT)
  if [ "$NUMBER_OF_TAGS_FOR_COMMIT" -gt 1 ]
  then
      echo "Current commit points to more than one tag: cannot continue"
      exit 2
  fi
  if [ $NUMBER_OF_TAGS_FOR_COMMIT -eq 0 ] # All changes are committed, but not tag
  then
      LAST_TAG_BY_VERSION=$( git tag --list '[vV]*' --sort=v:refname | tail -1 )
      MINOR_VERSION=$(echo $LAST_TAG_BY_VERSION | cut -f3 -d.)
      MAJOR_VERSION=$(echo $LAST_TAG_BY_VERSION | cut -f2 -d.)
      NEXT_MINOR_VERSION=$(($MINOR_VERSION + 1))
      NEXT_MAJOR_VERSION=$(($MAJOR_VERSION + 1))
      TAG_FOR_CURRENT_COMMIT="0.$MAJOR_VERSION.$NEXT_MINOR_VERSION"
      echo "Current commit does not have any matching tag: Creating a new tag to the next minor version: $TAG_FOR_CURRENT_COMMIT"
      echo "If you want to increment major version, re-run this script after creating the tag manually using:git tag 0.$NEXT_MAJOR_VERSION.0"
  fi
  if [ "$NUMBER_OF_TAGS_FOR_COMMIT" -eq 1 ]; then
      echo "Current commit points existing tag: $TAG_FOR_CURRENT_COMMIT"
  fi
fi

REDHAT_RELEASE_REPO_NAME=pkgs.devel.redhat.com
OPERATOR_CONTAINER_REPO_NAME=jenkins-operator
OPERATOR_BUNDLE_REPO_NAME=jenkins-operator-bundle

BASE_REPO_DIR=$GOPATH/src/$REDHAT_RELEASE_REPO_NAME/2
OPERATOR_CONTAINER_IMAGE_DIR=$BASE_REPO_DIR/$OPERATOR_CONTAINER_REPO_NAME
OPERATOR_BUNDLE_IMAGE_DIR=$BASE_REPO_DIR/$OPERATOR_BUNDLE_REPO_NAME

REDHAT_RELEASE_REPO=ssh://$REDHAT_RELEASE_REPO_NAME/containers
OPERATOR_CONTAINER_IMAGE_REPO_GIT="${REDHAT_RELEASE_REPO}/${OPERATOR_CONTAINER_REPO_NAME}"
OPERATOR_BUNDLE_IMAGE_REPO_GIT="${REDHAT_RELEASE_REPO}/${OPERATOR_BUNDLE_REPO_NAME}"

mkdir -p "${BASE_REPO_DIR}"
clone_if_not_existent $OPERATOR_CONTAINER_IMAGE_DIR $OPERATOR_CONTAINER_IMAGE_REPO_GIT
clone_if_not_existent $OPERATOR_BUNDLE_IMAGE_DIR    $OPERATOR_BUNDLE_IMAGE_REPO_GIT

echo "Copying release files into redhat repositories for ${OPERATOR_CONTAINER_REPO_NAME}"
rm  -fr $OPERATOR_CONTAINER_IMAGE_DIR/*
cp -fr ./redhat-release/$OPERATOR_CONTAINER_REPO_NAME/. $OPERATOR_CONTAINER_IMAGE_DIR


echo "Copying release files into redhat repositories for ${OPERATOR_BUNDLE_REPO_NAME}"
rm -fr $OPERATOR_BUNDLE_IMAGE_DIR/*
cp -fr ./redhat-release/$OPERATOR_BUNDLE_REPO_NAME/. $OPERATOR_BUNDLE_IMAGE_DIR
