// Copyright Contributors to the Open Cluster Management project

package apis

import (
	v1 "github.com/open-cluster-management/multiclusterhub-operator/pkg/apis/operator/v1"
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes, v1.SchemeBuilder.AddToScheme)
}
