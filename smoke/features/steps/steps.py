import os
import time
import logging
import urllib3
from behave import given, when, then
from datetime import date
from pyshould import should
from kubernetes import config, client
from smoke.features.steps.openshift import Openshift
from smoke.features.steps.project import Project

'''
If we need to install an operator manually using the cli 
- ensure your catalog source is installed
- create an OperatorGroup
- create the Subscription object
'''

# Path to the yaml files
scripts_dir = os.getenv('OUTPUT_DIR')
# jenkins_crd = './manifests/jenkins-operator/0.7.0/'
catalogsource = './smoke/samples/catalog-source.yaml'
operatorgroup = os.path.join(scripts_dir,'operator-group.yaml')
subscription = os.path.join(scripts_dir,'subscription.yaml')
jenkins = os.path.join(scripts_dir,'jenkins.yaml')
deploy_pod = "jenkins-1-deploy"
samplebclst = ['sample-pipeline','nodejs-mongodb-example']
samplepipeline = "https://raw.githubusercontent.com/openshift/origin/master/examples/jenkins/pipeline/samplepipeline.yaml"
# variables needed to get the resource status
current_project = ''
config.load_kube_config()
v1 = client.CoreV1Api()
oc = Openshift()

podStatus = {}


# STEP


@given(u'Project "{project_name}" is used')
def given_project_is_used(context, project_name):
    project = Project(project_name)
    global current_project
    current_project = project_name
    context.current_project = current_project
    context.oc = oc
    if not project.is_present():
        print("Project is not present, creating project: {}...".format(project_name))
        project.create() | should.be_truthy.desc(
            "Project {} is created".format(project_name))
    print("Project {} is created!!!".format(project_name))
    context.project = project


# STEP
@given(u'Project [{project_env}] is used')
def given_namespace_from_env_is_used(context, project_env):
    env = os.getenv(project_env)
    assert env is not None, f"{project_env} environment variable needs to be set"
    print(f"{project_env} = {env}")
    given_project_is_used(context, env)


@given(u'we have a openshift cluster')
def loginCluster(context):
    print("Using [{}]".format(current_project))


@when(u'we create the catalog source using catalog-source.yaml')
def createCatalogsource(context):
    res = oc.oc_create_from_yaml(catalogsource)
    print(res)


@then(u'we create operator group using operator-group.yaml')
def createOperatorgroup(context):
    res = oc.oc_create_from_yaml(operatorgroup)
    print(res)


@then(u'we create subscription using subscriptions.yaml')
def createSubsObject(context):
    res = oc.oc_create_from_yaml(subscription)
    print(res)


@then(u'we check for the csv and csv version')
def verifycsv(context):
    print('---> Getting the resources')
    time.sleep(10)
    if not 'jenkins-operator.0.0.0' in oc.search_resource_in_namespace('csv','jenkins-operator.0.0.0',current_project):
        raise AssertionError
    else:
        res = oc.search_resource_in_namespace('csv','jenkins-operator.0.0.0',current_project)
        print(res)


@then(u'we check for the operator group')
def verifyoperatorgroup(context):
    if not 'jenkins-operator' in oc.search_resource_in_namespace('operatorgroup','jenkins-operator',current_project):
        raise AssertionError
    else:
        res = oc.search_resource_in_namespace('operatorgroup','jenkins-operator',current_project)
        print(res)


@then(u'we check for the subscription')
def verifysubs(context):
    if not 'jenkins-operator' in oc.search_resource_in_namespace('subs','jenkins-operator',current_project):
        raise AssertionError
    else:
        res = oc.search_resource_in_namespace('subs','jenkins-operator',current_project)
        print(res)


@then(u'we check for the operator pod')
def verifyoperatorpod(context):
    print('---> checking operator pod status')
    context.v1 = v1
    pods = v1.list_namespaced_pod(current_project)
    for i in pods.items:
        print("Getting pod list")
        podStatus[i.metadata.name] = i.status.phase
        print('---> Validating...')
        if not i.metadata.name in oc.search_pod_in_namespace(i.metadata.name,current_project):
            raise AssertionError

    print('waiting to get pod status')
    time.sleep(10)
    for pod in podStatus.keys():
        status = podStatus[pod]
        if 'Running' in status:
            print(pod)
            print(podStatus[pod])
        else:
            raise AssertionError

@given(u'Jenkins operator is installed')
def verifyoperator(context):
    verifyoperatorpod(context)
    

@when(u'we create the jenkins instance using jenkins.yaml')
def createinstance(context):
    res = oc.oc_create_from_yaml(jenkins)
    print(res)


@then(u'We check for the jenkins-example pod status')
def checkjenkinspod(context):
    verifyoperatorpod(context)

@then(u'We check for the route')
def checkroute(context):
    operator_name = 'jenkins-simple'
    time.sleep(10)
    route = oc.get_route_host(operator_name,current_project)
    url = 'http://'+str(route)
    print('--->App url:')
    print(url)
    
    if len(url) <= 0:
        raise AssertionError

@given(u'The jenkins pod is up and runnning')
def checkJenkins(context):
    time.sleep(10)
    podStatus = {}
    status = ""
    pods = v1.list_namespaced_pod(current_project)
    for i in pods.items:
        print("Getting pod list")
        print(i.status.pod_ip)
        print(i.metadata.name)
        print(i.status.phase)
        podStatus[i.metadata.name] = i.status.phase
    for pod in podStatus.keys():
        status = podStatus[pod]
        if 'Running' in status:
            print("still checking pod status")
            print(pod)
            print(podStatus[pod])
        elif 'Succeeded' in status:
            print("checking pod status")
            print(pod)
            print(podStatus[pod])
        else:
            raise AssertionError


@when(u'The user enters new-app command with sample-pipeline')
def createPipeline(context):
    # bclst = ['sample-pipeline','nodejs-mongodb-example']
    res = oc.new_app_from_file(samplepipeline,current_project)
    for item, value in enumerate(samplebclst):
        if 'sample-pipeline' in oc.search_resource_in_namespace('bc',value, current_project):
            print('Buildconfig sample-pipeline created')
        elif 'nodejs-mongodb-example' in oc.search_resource_in_namespace('bc',value,current_project):
            print('Buildconfig nodejs-mongodb-example created')
        else:
            raise AssertionError
    print(res)


@then(u'Trigger the build using oc start-build')
def startbuild(context):
    for item,value in enumerate(samplebclst):
        res = oc.start_build(value,current_project)
        if not value in res:
            raise AssertionError
        else:
            print(res)


@then(u'nodejs-mongodb-example pod must come up')
def check_app_pod(context):
    time.sleep(120)
    podStatus = {}
    podSet = set()
    bcdcSet = set()
    pods = v1.list_namespaced_pod(current_project)
    for i in pods.items:
        podStatus[i.metadata.name] = i.status.phase
        podSet.add(i.metadata.name)
    
    for items in podSet:
        if 'build' in items:
           bcdcSet.add(items)
        elif 'deploy' in items:
            bcdcSet.add(items)

    app_pods = podSet.difference(bcdcSet)
    for items in app_pods:
        print('Getting pods')
        print(items)
    
    for items in app_pods:
        for pod in podStatus.keys():
            status = podStatus[items]
            if not 'Running' in status:
                raise AssertionError
    print('---> App pods are ready')

@then(u'route nodejs-mongodb-example must be created and be accessible')
def connectApp(context):
    print('Getting application route/url')
    app_name = 'nodejs-mongodb-example'
    time.sleep(30)
    route = oc.get_route_host(app_name,current_project)
    url = 'http://'+str(route)
    print('--->App url:')
    print(url)
    http = urllib3.PoolManager()
    res = http.request('GET', url)
    connection_status = res.status
    if connection_status == 200:
        print('---> Application is accessible via the route')
        print(url)
    else:
        raise Exception