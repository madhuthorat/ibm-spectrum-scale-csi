package csiscaleoperator

import (
	"fmt"

	csiv1 "github.com/IBM/ibm-spectrum-scale-csi/api/v1"
	"github.com/IBM/ibm-spectrum-scale-csi/controllers/config"
	csiversion "github.com/IBM/ibm-spectrum-scale-csi/controllers/version"
	"k8s.io/apimachinery/pkg/labels"
)

// CSIScaleOperator is the wrapper for csiv1.CSIScaleOperator type
type CSIScaleOperator struct {
	*csiv1.CSIScaleOperator
	//	ServerVersion string
}

// New returns a wrapper for csiv1.CSIScaleOperator
func New(c *csiv1.CSIScaleOperator) *CSIScaleOperator {
	return &CSIScaleOperator{
		CSIScaleOperator: c,
	}
}

// Unwrap returns the csiv1.CSIScaleOperator object
func (c *CSIScaleOperator) Unwrap() *csiv1.CSIScaleOperator {
	return c.CSIScaleOperator
}

// GetLabels returns all the labels to be set on all resources
//func (c *CSIScaleOperator) GetLabels() labels.Set {
func (c *CSIScaleOperator) GetLabels() map[string]string {
	labels := labels.Set{
		"app.kubernetes.io/name":       config.ProductName,
		"app.kubernetes.io/instance":   c.Name,
		"app.kubernetes.io/version":    csiversion.Version,
		"app.kubernetes.io/managed-by": config.Name,
		"csi":                          "ibm",
		"product":                      config.ProductName,
		"release":                      fmt.Sprintf("v%s", csiversion.Version),
	}

	if c.Labels != nil {
		for k, v := range c.Labels {
			if !labels.Has(k) {
				labels[k] = v
			}
		}
	}

	return labels
}

// GetAnnotations returns all the annotations to be set on all resources
//func (c *CSIScaleOperator) GetAnnotations(daemonSetRestartedKey string, daemonSetRestartedValue string) labels.Set {
func (c *CSIScaleOperator) GetAnnotations(daemonSetRestartedKey string, daemonSetRestartedValue string) map[string]string {
	//func (c *CSIScaleOperator) GetAnnotations() map[string]string {
	labels := labels.Set{
		"productID":      config.ProductName,
		"productName":    config.ProductName,
		"productVersion": csiversion.Version,
	}

	if c.Annotations != nil {
		for k, v := range c.Annotations {
			if !labels.Has(k) {
				labels[k] = v
			}
		}
	}

	if !labels.Has(daemonSetRestartedKey) && daemonSetRestartedKey != "" {
		labels[daemonSetRestartedKey] = daemonSetRestartedValue
	}

	return labels
}

// GetSelectorLabels returns labels used in label selectors
func (c *CSIScaleOperator) GetSelectorLabels(component string) labels.Set {
	return labels.Set{
		"app.kubernetes.io/component": component,
	}
}

func (c *CSIScaleOperator) GetCSIControllerSelectorLabels() labels.Set {
	return c.GetSelectorLabels(config.CSIController.String())
}

func (c *CSIScaleOperator) GetCSINodeSelectorLabels() labels.Set {
	return c.GetSelectorLabels(config.CSINode.String())
}

func (c *CSIScaleOperator) GetCSIControllerPodLabels() labels.Set {
	return labels.Merge(c.GetLabels(), c.GetCSIControllerSelectorLabels())
}

func (c *CSIScaleOperator) GetCSINodePodLabels() labels.Set {
	return labels.Merge(c.GetLabels(), c.GetCSINodeSelectorLabels())
}

func (c *CSIScaleOperator) GetCSIControllerImage() string {
	if c.Spec.ControllerTag == "" {
		return c.Spec.ControllerRepository
	}
	return c.Spec.ControllerRepository + ":" + c.Spec.ControllerTag
}

func (c *CSIScaleOperator) GetCSINodeImage() string {
	if c.Spec.Node.Tag == "" {
		return c.Spec.Node.Repository
	}
	return c.Spec.Node.Repository + ":" + c.Spec.Node.Tag
}

func (c *CSIScaleOperator) GetDefaultSidecarImageByName(name string) string {
	if sidecar, found := config.DefaultSidecarsByName[name]; found {
		return fmt.Sprintf("%s:%s", sidecar.Repository, sidecar.Tag)
	}
	return ""
}
