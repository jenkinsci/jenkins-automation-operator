module github.com/jenkinsci/jenkins-automation-operator

go 1.13

require (
	github.com/bndr/gojenkins v0.0.0-20181125150310-de43c03cf849
	github.com/coreos/prometheus-operator v0.0.0-00010101000000-000000000000
	github.com/daixiang0/gci v0.2.4 // indirect
	github.com/docker/distribution v2.7.1+incompatible
	github.com/elazarl/goproxy v0.0.0-20190711103511-473e67f1d7d2 // indirect
	github.com/elazarl/goproxy/ext v0.0.0-20190711103511-473e67f1d7d2 // indirect
	github.com/emersion/go-sasl v0.0.0-20190817083125-240c8404624e // indirect
	github.com/emersion/go-smtp v0.11.2 // indirect
	github.com/go-logr/logr v0.1.0
	github.com/go-logr/zapr v0.1.1
	github.com/golang/mock v1.3.1
	github.com/golangci/golangci-lint v1.26.0 // indirect
	github.com/google/go-cmp v0.5.2 // indirect
	github.com/mailgun/mailgun-go/v3 v3.6.0 // indirect
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.2
	github.com/openshift/api v3.9.1-0.20190924102528-32369d4db2ad+incompatible
	github.com/openshift/custom-resource-status v0.0.0-20200602122900-c002fd1547ca
	github.com/operator-framework/api v0.3.13 // indirect
	github.com/operator-framework/operator-lib v0.1.0
	github.com/operator-framework/operator-sdk v1.0.1 // indirect
	github.com/pkg/errors v0.9.1
	github.com/robfig/cron v1.2.0 // indirect
	github.com/sonatard/noctx v0.0.1 // indirect
	github.com/spf13/pflag v1.0.5
	github.com/ssgreg/nlreturn v1.0.5 // indirect
	github.com/stretchr/testify v1.5.1
	go.uber.org/zap v1.14.1
	golang.org/x/lint v0.0.0-20200302205851-738671d3881b // indirect
	golang.org/x/net v0.0.0-20200822124328-c89045814202 // indirect
	golang.org/x/tools v0.0.0-20201002184944-ecd9fd270d5d // indirect
	gopkg.in/alexcesaro/quotedprintable.v3 v3.0.0-20150716171945-2caba252f4dc // indirect
	gopkg.in/gomail.v2 v2.0.0-20160411212932-81ebce5c23df // indirect
	k8s.io/api v0.18.6
	k8s.io/apimachinery v0.18.6
	k8s.io/cli-runtime v0.18.2 // indirect
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/kubectl v0.18.2
	k8s.io/kubernetes v1.13.0
	k8s.io/utils v0.0.0-20190801114015-581e00157fb1
	mvdan.cc/gofumpt v0.0.0-20200927160801-5bfeb2e70dd6 // indirect
	sigs.k8s.io/controller-runtime v0.6.2
)

// Pinned to kubernetes-1.16.2
replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v12.2.0+incompatible
	k8s.io/api => k8s.io/api v0.0.0-20191016110408-35e52d86657a
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20191016113550-5357c4baaf65
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20191004115801-a2eda9f80ab8
	k8s.io/apiserver => k8s.io/apiserver v0.0.0-20191016112112-5190913f932d
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.0.0-20191016114015-74ad18325ed5
	k8s.io/client-go => k8s.io/client-go v0.0.0-20191016111102-bec269661e48
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.0.0-20191016115326-20453efc2458
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.0.0-20191016115129-c07a134afb42
	k8s.io/component-base => k8s.io/component-base v0.0.0-20191016111319-039242c015a9
	k8s.io/cri-api => k8s.io/cri-api v0.0.0-20190828162817-608eb1dad4ac
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.0.0-20191016115521-756ffa5af0bd
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.0.0-20191016112429-9587704a8ad4
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.0.0-20191016114939-2b2b218dc1df
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.0.0-20191016114407-2e83b6f20229
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.0.0-20191016114748-65049c67a58b
	k8s.io/kubectl => k8s.io/kubectl v0.0.0-20191016120415-2ed914427d51
	k8s.io/kubelet => k8s.io/kubelet v0.0.0-20191016114556-7841ed97f1b2
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.0.0-20191016115753-cf0698c3a16b
	k8s.io/metrics => k8s.io/metrics v0.0.0-20191016113814-3b1a734dba6e
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.0.0-20191016112829-06bb3c9d77c9
)

replace (
	github.com/coreos/prometheus-operator => github.com/coreos/prometheus-operator v0.35.1
	github.com/docker/docker => github.com/moby/moby v0.7.3-0.20190826074503-38ab9da00309
	github.com/operator-framework/operator-sdk => github.com/operator-framework/operator-sdk v1.0.1
	k8s.io/code-generator => k8s.io/code-generator v0.0.0-20181117043124-c2090bec4d9b
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20180711000925-0cf8f7e6ed1d
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.4.0
	sigs.k8s.io/controller-tools => sigs.k8s.io/controller-tools v0.1.11-0.20190411181648-9d55346c2bde
)
