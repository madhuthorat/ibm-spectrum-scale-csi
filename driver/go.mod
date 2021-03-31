module github.com/IBM/ibm-spectrum-scale-csi/driver

go 1.15

require (
	github.com/container-storage-interface/spec v1.1.0
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/golang/protobuf v1.3.2
	golang.org/x/net v0.0.0-20191028085509-fe3aa8a45271
	google.golang.org/genproto v0.0.0-20190502173448-54afdca5d873 // indirect
	google.golang.org/grpc v1.24.0
        k8s.io/apimachinery v0.21.0-alpha.0
        k8s.io/kubernetes v1.20.0
        k8s.io/mount-utils v0.20.0 // indirect
        k8s.io/utils v0.0.0-20201110183641-67b214c5f920
)

replace k8s.io/api => k8s.io/api v0.20.0

replace k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.20.0

replace k8s.io/apimachinery => k8s.io/apimachinery v0.21.0-alpha.0

replace k8s.io/apiserver => k8s.io/apiserver v0.20.0

replace k8s.io/cli-runtime => k8s.io/cli-runtime v0.20.0

replace k8s.io/client-go => k8s.io/client-go v0.20.0

replace k8s.io/cloud-provider => k8s.io/cloud-provider v0.20.0

replace k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.20.0

replace k8s.io/code-generator => k8s.io/code-generator v0.20.1-rc.0

replace k8s.io/component-base => k8s.io/component-base v0.20.0

replace k8s.io/component-helpers => k8s.io/component-helpers v0.20.0

replace k8s.io/controller-manager => k8s.io/controller-manager v0.20.0

replace k8s.io/cri-api => k8s.io/cri-api v0.20.1-rc.0

replace k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.20.0

replace k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.20.0

replace k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.20.0

replace k8s.io/kube-proxy => k8s.io/kube-proxy v0.20.0

replace k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.20.0

replace k8s.io/kubectl => k8s.io/kubectl v0.20.0

replace k8s.io/kubelet => k8s.io/kubelet v0.20.0

replace k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.20.0

replace k8s.io/metrics => k8s.io/metrics v0.20.0

replace k8s.io/mount-utils => k8s.io/mount-utils v0.20.1-rc.0

replace k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.20.0

replace k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.20.0

replace k8s.io/sample-controller => k8s.io/sample-controller v0.20.0

