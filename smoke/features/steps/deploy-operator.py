import time
import logging
from behave import given, when, then
from datetime import date
from pyshould import should
from kubernetes import config, client
from smoke.features.steps.openshift import Openshift
from smoke.features.steps.project import Project


def get_filename_datetime():
    # Use current date to get a text file name.
    return "" + str(date.today()) + ".txt"


# Get full path for writing.
file_name = get_filename_datetime()
path = "./smoke/logs-" + file_name
crd_path = "./deploy/crds/jenkins_all_crd.yaml"
template_path = "./smoke/jenkins-operator-template.yaml"
cr_path = "./deploy/crds/openshift_jenkins_v1alpha2_jenkins_cr.yaml"
logging.basicConfig(filename=path, format='%(asctime)s: %(levelname)s: %(message)s', datefmt='%m/%d/%Y %I:%M:%S %p')
logger = logging.getLogger()
logger.setLevel(logging.INFO)

project_name = 'jenkins-test'
paramfile = './smoke/templates.params'
oc = Openshift()


@given(u'We have a openshift cluster')
def loginCluster(context):
    project = Project(project_name)
    context.oc = oc
    if not project.is_present():
        logger.info("Project is not present, creating project: {}...".format(project_name))
        project.create() | should.be_truthy.desc("Project {} is created".format(project_name))

    logger.info(f'Project {project_name} is created!!!')
    context.project = project
    


@then(u'we build the jenkins operator image')
def step_impl(context):
    pass

@then(u'we push to openshift internal registry')
def step_impl(context):
    pass

@when(u'the resources are created using the crd')
def createResources(context):
    logger.info("using crd to create the required resources")
    res = oc.oc_create_from_yaml(crd_path)
    logger.info(res)
    

@then(u'We create template from yaml')
def createTemplate(context):
    res = oc.oc_create_from_yaml(template_path)
    time.sleep(20)
    if not 'jenkins-operator-template' in res:
        raise AssertionError


@then(u'Apply template with oc new-app')
def createOperator(context):
    template_name = 'jenkins-operator-template'
    res = oc.new_app_with_params(template_name,project_name,paramfile)


@then(u'Check for pod creation and state')
def step_impl(context):
    pass


@then(u'Check health of the operator')
def step_impl(context):
    pass

@given(u'Jenkins operator is running')
def step_impl(context):
    raise NotImplementedError(u'STEP: Given Jenkins operator is running')


@when(u'we create the jenkins cr')
def step_impl(context):
    raise NotImplementedError(u'STEP: When we create the jenkins cr')


@then(u'we check the jenkins pod health')
def step_impl(context):
    raise NotImplementedError(u'STEP: Then we check the jenkins pod health')
