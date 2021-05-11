'''
before_step(context, step), after_step(context, step)
    These run before and after every step.
    The step passed in is an instance of Step.
before_feature(context, feature), after_feature(context, feature)
    These run before and after each feature file is exercised.
    The feature passed in is an instance of Feature.
before_all(context), after_all(context)
    These run before and after the whole shooting match.
'''

import subprocess
from pyfiglet import Figlet
from pyshould import should
from clint.textui import puts,colored, indent


f = Figlet(font='starwars', width=100)

def before_feature(_context,_feature):
    '''
    before_feature(context, feature), after_feature(context, feature)
    These run before and after each feature is run.
    '''
    print(colored.red(f.renderText('Jenkins Operator')))
    print(colored.yellow("checking cluster environment for {} scenario".format(_feature)))

def before_scenario(_context, _scenario):
    '''
    before_scenario(context, scenario), after_scenario(context, scenario)
    These run before and after each scenario is run.
    The scenario passed in is an instance of Scenario.
'''
    print(colored.yellow("Checking cluster environment for {} scenario".format(_scenario)))
    code, output = subprocess.getstatusoutput('oc get project default')
    print(colored.yellow("[CODE] {}".format(code)))
    print(colored.yellow("[CMD] {}".format(output)))
    code | should.be_equal_to(0)
    print(colored.green("*************Connected to cluster*************"))
