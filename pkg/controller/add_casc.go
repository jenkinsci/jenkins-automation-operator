package controller

import (
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/casc"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, casc.Add)
}
