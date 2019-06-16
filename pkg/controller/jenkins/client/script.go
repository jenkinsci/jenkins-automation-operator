package client

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/bndr/gojenkins"
	"github.com/pkg/errors"
)

func (jenkins *jenkins) ExecuteScript(script string) (string, error) {
	now := time.Now().Unix()
	verifier := fmt.Sprintf("verifier-%d", now)
	return jenkins.executeScript(script, verifier)
}

func (jenkins *jenkins) executeScript(script string, verifier string) (string, error) {
	output := ""
	fullScript := fmt.Sprintf("%s\nprint println('%s')", script, verifier)
	parameters := map[string]string{"script": fullScript}

	ar := gojenkins.NewAPIRequest("POST", "/scriptText", nil)
	if err := jenkins.Requester.SetCrumb(ar); err != nil {
		return output, err
	}
	ar.SetHeader("Content-Type", "application/x-www-form-urlencoded")
	ar.Suffix = ""

	r, err := jenkins.Requester.Do(ar, &output, parameters)
	if err != nil {
		return "", errors.Wrapf(err, "couldn't execute groovy script, logs '%s'", output)
	}

	if r.StatusCode != http.StatusOK {
		return output, errors.Errorf("invalid status code '%d', logs '%s'", r.StatusCode, output)
	}

	if !strings.Contains(output, verifier) {
		return output, errors.Errorf("script execution failed, logs '%s'", output)
	}

	return output, nil
}
