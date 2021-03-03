#!/bin/sh

#-----------------------------------------------------------------------------
# Global Variables
#-----------------------------------------------------------------------------
export V_FLAG=-v
export OUTPUT_DIR=$(pwd)/out
export LOGS_DIR=${OUTPUT_DIR}/logs
export GOLANGCI_LINT_BIN=${OUTPUT_DIR}/golangci-lint
export PYTHON_VENV_DIR=${OUTPUT_DIR}/venv3
# -- Variables for smoke tests
export TEST_SMOKE_ARTIFACTS=/tmp/artifacts

# -- Setting up the venv
python3 -m venv ${PYTHON_VENV_DIR}
${PYTHON_VENV_DIR}/bin/pip install --upgrade setuptools
${PYTHON_VENV_DIR}/bin/pip install --upgrade pip
# -- Generating a new namespace name
echo "test-namespace-$(uuidgen | tr '[:upper:]' '[:lower:]' | head -c 8)" > ${OUTPUT_DIR}/test-namespace
export TEST_NAMESPACE=$(cat ${OUTPUT_DIR}/test-namespace)
echo "Assigning value to varibale TEST_NAMESPACE:"${TEST_NAMESPACE}
# -- Do clean up & create namespace
echo "Starting cleanup"
kubectl delete namespace ${TEST_NAMESPACE} --timeout=45s --wait
kubectl create namespace ${TEST_NAMESPACE}
mkdir -p ${LOGS_DIR}/smoke-tests-logs
mkdir -p ${OUTPUT_DIR}/smoke-tests-output
export TEST_SMOKE_OUTPUT_DIR=${OUTPUT_DIR}/smoke
echo "Logs directory created at "{$LOGS_DIR/smoke}

# -- Set namespace for subscription.yaml & operator-group.yaml file , set the TEST_NAMESPACE as current project

sed "s|PROJECT|${TEST_NAMESPACE}|g" smoke/samples/subscription.yaml > $OUTPUT_DIR/subscription.yaml
sed "s|PROJECT|${TEST_NAMESPACE}|g" smoke/samples/operator-group.yaml > $OUTPUT_DIR/operator-group.yaml
sed "s|PROJECT|${TEST_NAMESPACE}|g" smoke/samples/jenkins.yaml > $OUTPUT_DIR/jenkins.yaml

# -- Setting the project
oc project ${TEST_NAMESPACE}

# -- Trigger the test
echo "Starting local Jenkins instance"
${PYTHON_VENV_DIR}/bin/pip install -q -r smoke/requirements.txt
echo "Running smoke tests"
TEST_NAMESPACE=${TEST_NAMESPACE}
${PYTHON_VENV_DIR}/bin/behave --junit --junit-directory ${TEST_SMOKE_OUTPUT_DIR} --no-capture --no-capture-stderr smoke/features
echo "Logs collected at "${TEST_SMOKE_OUTPUT_DIR}

