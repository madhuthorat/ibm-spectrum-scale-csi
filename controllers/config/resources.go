package config

import "fmt"

// ResourceName is the type for aliasing resources that will be created.
type ResourceName string

func (rn ResourceName) String() string {
	return string(rn)
}

const (
	CSIController                         ResourceName = "csi-controller"
	CSINode                               ResourceName = "csi-node"
	NodeAgent                             ResourceName = "ibm-node-agent"
	CSIControllerServiceAccount           ResourceName = "csi-controller-sa"
	CSINodeServiceAccount                 ResourceName = "csi-node-sa"
	ExternalProvisionerClusterRole        ResourceName = "external-provisioner-clusterrole"
	ExternalProvisionerClusterRoleBinding ResourceName = "external-provisioner-clusterrolebinding"
	ExternalAttacherClusterRole           ResourceName = "external-attacher-clusterrole"
	ExternalAttacherClusterRoleBinding    ResourceName = "external-attacher-clusterrolebinding"
	ExternalSnapshotterClusterRole        ResourceName = "external-snapshotter-clusterrole"
	ExternalSnapshotterClusterRoleBinding ResourceName = "external-snapshotter-clusterrolebinding"
	ExternalResizerClusterRole            ResourceName = "external-resizer-clusterrole"
	ExternalResizerClusterRoleBinding     ResourceName = "external-resizer-clusterrolebinding"
	CSIControllerSCCClusterRole           ResourceName = "csi-controller-scc-clusterrole"
	CSIControllerSCCClusterRoleBinding    ResourceName = "csi-controller-scc-clusterrolebinding"
	CSINodeSCCClusterRole                 ResourceName = "csi-node-scc-clusterrole"
	CSINodeSCCClusterRoleBinding          ResourceName = "csi-node-scc-clusterrolebinding"
)

// GetNameForResource returns the name of a resource for a CSI driver
func GetNameForResource(name ResourceName, driverName string) string {
	switch name {
	case CSIController:
		return fmt.Sprintf("%s-controller", driverName)
	case CSINode:
		return fmt.Sprintf("%s-node", driverName)
	case CSIControllerServiceAccount:
		return fmt.Sprintf("%s-controller-sa", driverName)
	case CSINodeServiceAccount:
		return fmt.Sprintf("%s-node-sa", driverName)
	default:
		return fmt.Sprintf("%s-%s", driverName, name)
	}
}
