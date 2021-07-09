package config

// Add a field here if it never changes, if it changes over time, put it to settings.go
const (
	APIGroup    = "csi.ibm.com"
	APIVersion  = "csi.ibm.com/v1"
	Name        = "ibm-spectrum-scale-csi-operator"
	DriverName  = "csi.ibm.com"
	ProductName = "ibm-spectrum-scale-csi-driver"
	Masterlabel = "node-role.kubernetes.io/master"

	NodeAgentRepository = "ibmcom/ibm-node-agent"

	ENVEndpoint    = "ENDPOINT"
	ENVNodeName    = "NODE_NAME"
	ENVKubeVersion = "KUBE_VERSION"

	CSINodeDriverRegistrar = "csi-node-driver-registrar"
	CSIProvisioner         = "csi-provisioner"
	CSIAttacher            = "csi-attacher"
	CSISnapshotter         = "csi-snapshotter"
	CSIResizer             = "csi-resizer"
	LivenessProbe          = "livenessprobe"

	ControllerSocketVolumeMountPath                       = "/var/lib/csi/sockets/pluginproxy/"
	NodeSocketVolumeMountPath                             = "/csi"
	ControllerLivenessProbeContainerSocketVolumeMountPath = "/csi"
	ControllerSocketPath                                  = "/var/lib/csi/sockets/pluginproxy/csi.sock"
	NodeSocketPath                                        = "/csi/csi.sock"
	NodeRegistrarSocketPath                               = "/var/lib/kubelet/plugins/csi.ibm.com/csi.sock"
	CSIEndpoint                                           = "unix:///var/lib/csi/sockets/pluginproxy/csi.sock"
	CSINodeEndpoint                                       = "unix:///csi/csi.sock"
)
