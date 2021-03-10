#!/bin/sh

. $(dirname $(realpath $0))/release-functions.sh
. $(dirname $(realpath $0))/colors.sh

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


echo "${blue}Updating container.yml for image and container to point to latest sources${reset}"

OPERATOR_NAME=jenkins-operator
OPERATOR_CONTAINER_REPO_NAME=jenkins-operator
OPERATOR_BUNDLE_REPO_NAME=jenkins-operator-bundle
JENKINS_OPERATOR_REPO=https://github.com/redhat-developer/$OPERATOR_CONTAINER_REPO_NAME


REDHAT_RELEASE_REPO_NAME=pkgs.devel.redhat.com
BASE_REPO_DIR=$GOPATH/src/$REDHAT_RELEASE_REPO_NAME/containers
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
cp -fr ./bundle/manifests $OPERATOR_BUNDLE_IMAGE_DIR
cp -fr ./bundle/metadata $OPERATOR_BUNDLE_IMAGE_DIR


JENKINS_SERVER_IMAGE_VERSION="v4.7"
JENKINS_BACKUP_IMAGE_VERSION="8.3"
JENKINS_BUILDER_IMAGE_VERSION="v1.3.1"
JENKINS_SIDECAR_IMAGE_VERSION="v1.0"

JENKINS_SERVER_IMAGE_NAME="registry.redhat.io/openshift4/ose-jenkins"
JENKINS_BACKUP_IMAGE_NAME="registry.redhat.io/ubi8/ubi-minimal"
JENKINS_BUILDER_IMAGE_NAME="quay.io/redhat-developer/openshift-jenkins-image-builder"
JENKINS_SIDECAR_IMAGE_NAME="quay.io/redhat-developer/jenkins-kubernetes-sidecar"
#TODO When dependent images are published, use these coordinates instead
#JENKINS_SIDECAR_IMAGE_NAME="registry.redhat.io/ocp-tools-4/jenkins-rhel8-sidecar"
#JENKINS_BUILDER_IMAGE_NAME="registry.redhat.io/ocp-tools-4/jenkins-rhel8-builder"

JENKINS_SERVER_IMAGE_URL="docker://$JENKINS_SERVER_IMAGE_NAME:$JENKINS_SERVER_IMAGE_VERSION"
JENKINS_BACKUP_IMAGE_URL="docker://$JENKINS_BACKUP_IMAGE_NAME:$JENKINS_BACKUP_IMAGE_VERSION"
JENKINS_BUILDER_IMAGE_URL="docker://$JENKINS_BUILDER_IMAGE_NAME:$JENKINS_BUILDER_IMAGE_VERSION"
JENKINS_SIDECAR_IMAGE_URL="docker://$JENKINS_SIDECAR_IMAGE_NAME:$JENKINS_SIDECAR_IMAGE_VERSION"


for file in $(find "${OPERATOR_BUNDLE_IMAGE_DIR}/" -type f -not -path '*/\.git/*') ; do
  sed -i.backup s+@@OPERATOR_COMMIT@@+$JENKINS_OPERATOR_COMMIT+g $file
  sed -i.backup s+@@OPERATOR_REPO@@+$JENKINS_OPERATOR_REPO+g $file
  sed -i.backup s+@@OPERATOR_VERSION@@+$TAG_FOR_CURRENT_COMMIT+g $file
  sed -i.backup s+@@OPERATOR_NAME@@+$OPERATOR_NAME+g $file 
done

for file in $(find "${OPERATOR_CONTAINER_IMAGE_DIR}/" -type f -not -path '*/\.git/*') ; do
  sed -i.backup s+@@OPERATOR_COMMIT@@+$JENKINS_OPERATOR_COMMIT+g $file
  sed -i.backup s+@@OPERATOR_REPO@@+$JENKINS_OPERATOR_REPO+g $file
  sed -i.backup s+@@OPERATOR_VERSION@@+$TAG_FOR_CURRENT_COMMIT+g $file
  sed -i.backup s+@@OPERATOR_NAME@@+$OPERATOR_NAME+g $file 
done

SKOPEO_OPTIONS="--override-os linux   --authfile $HOME/.docker/config.json"
echo "Setting relatedImages and required envVars in Red Hat bundle manifest"
export JENKINS_SERVER_IMAGE="$JENKINS_SERVER_IMAGE_NAME@$(   skopeo inspect $SKOPEO_OPTIONS $JENKINS_SERVER_IMAGE_URL  | jq -r '.Digest')"
echo "-- Using $JENKINS_SERVER_IMAGE_URL as $JENKINS_SERVER_IMAGE"
export JENKINS_BACKUP_IMAGE="$JENKINS_BACKUP_IMAGE_NAME@$(    skopeo inspect $SKOPEO_OPTIONS $JENKINS_BACKUP_IMAGE_URL  | jq -r '.Digest')"
echo "-- Using $JENKINS_BACKUP_IMAGE_URL as $JENKINS_BACKUP_IMAGE"
export JENKINS_BUILDER_IMAGE="$JENKINS_BUILDER_IMAGE_NAME@$( skopeo inspect $SKOPEO_OPTIONS $JENKINS_BUILDER_IMAGE_URL | jq -r '.Digest')"
echo "-- Using $JENKINS_BUILDER_IMAGE_URL as $JENKINS_BUILDER_IMAGE"
export JENKINS_SIDECAR_IMAGE="$JENKINS_SIDECAR_IMAGE_NAME@$( skopeo inspect $SKOPEO_OPTIONS $JENKINS_SIDECAR_IMAGE_URL | jq -r '.Digest')"
echo "-- Using $JENKINS_SIDECAR_IMAGE_URL as $JENKINS_SIDECAR_IMAGE"

BUNDLE_FILE="$OPERATOR_BUNDLE_IMAGE_DIR/manifests/jenkins-operator.clusterserviceversion.yaml"

yq -i eval '(.spec.install.spec.deployments[].spec.template.spec.containers[] | select(.name=="manager").image) = "REPLACE_IMAGE"' $BUNDLE_FILE
yq -i eval '(.spec.install.spec.deployments[].spec.template.spec.containers[] | select(.name=="manager").env) += [{ "name": "OPERATOR_NAME","value":"jenkins-operator" }]' $BUNDLE_FILE
yq -i eval '(.spec.install.spec.deployments[].spec.template.spec.containers[] | select(.name=="manager").env)  += [{ "name": "POD_NAME","valueFrom": { "fieldRef": { "fieldPath": "metadata.name"}}}]' $BUNDLE_FILE
yq -i eval '(.spec.install.spec.deployments[].spec.template.spec.containers[] | select(.name=="manager").env) += [{ "name": "WATCH_NAMESPACE","valueFrom": { "fieldRef": { "fieldPath": "metadata.annotations['olm.targetNamespaces']"}}}]' $BUNDLE_FILE

yq -i eval '(.spec.install.spec.deployments[].spec.template.spec.containers[] | select(.name=="manager").env)  += [{ "name":"JENKINS_OPERATOR_IMAGE","value":"REPLACE_IMAGE"}]' $BUNDLE_FILE
yq -i eval '(.spec.install.spec.deployments[].spec.template.spec.containers[] | select(.name=="manager").env)  += [{ "name":"JENKINS_SERVER_IMAGE","value":strenv(JENKINS_SERVER_IMAGE)}]'   $BUNDLE_FILE
yq -i eval '(.spec.install.spec.deployments[].spec.template.spec.containers[] | select(.name=="manager").env)  += [{ "name":"JENKINS_BUILDER_IMAGE","value": strenv(JENKINS_BUILDER_IMAGE)}]'  $BUNDLE_FILE
yq -i eval '(.spec.install.spec.deployments[].spec.template.spec.containers[] | select(.name=="manager").env)  += [{ "name":"JENKINS_SIDECAR_IMAGE","value":strenv(JENKINS_SIDECAR_IMAGE)}]'  $BUNDLE_FILE
yq -i eval '(.spec.install.spec.deployments[].spec.template.spec.containers[] | select(.name=="manager").env)  += [{ "name":"JENKINS_BACKUP_IMAGE","value": strenv(JENKINS_BACKUP_IMAGE)}]'   $BUNDLE_FILE

yq -i eval '.spec.relatedImages += [{ "name": "jenkins-operator-image" , "image" : "REPLACE_IMAGE" }]' $BUNDLE_FILE
yq -i eval  '.spec.relatedImages += [{ "name": "jenkins-server-image"  , "image" : strenv(JENKINS_SERVER_IMAGE)  }]' $BUNDLE_FILE
yq -i eval  '.spec.relatedImages += [{ "name": "jenkins-builder-image" , "image" : strenv(JENKINS_BUILDER_IMAGE) }]' $BUNDLE_FILE
yq -i eval  '.spec.relatedImages += [{ "name": "jenkins-sidecar-image" , "image" : strenv(JENKINS_SIDECAR_IMAGE) }]' $BUNDLE_FILE
yq -i eval  '.spec.relatedImages += [{ "name": "jenkins-backup-image"  , "image" : strenv(JENKINS_BACKUP_IMAGE)  }]' $BUNDLE_FILE



echo ${yellow}Release preparation finished${reset}
echo ${blue}Change to directory ${yellow}$OPERATOR_BUNDLE_IMAGE_DIR${blue} and commit/push the generated files to continue release on CPaaS${reset}
echo ${blue}Change to directory ${yellow}$OPERATOR_CONTAINER_IMAGE_DIR${blue} and commit/push the generated files to continue release on CPaaS${reset}
