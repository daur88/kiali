package destinationrules

import (
	"strconv"

	"github.com/kiali/kiali/config"
	"github.com/kiali/kiali/kubernetes"
	"github.com/kiali/kiali/models"
)

type NoDestinationChecker struct {
	Namespace       string
	WorkloadList    models.WorkloadList
	DestinationRule kubernetes.IstioObject
}

// Check parses the DestinationRule definitions and verifies that they point to an existing service, including any subset definitions
func (n NoDestinationChecker) Check() ([]*models.IstioCheck, bool) {
	valid := true
	validations := make([]*models.IstioCheck, 0)

	if host, ok := n.DestinationRule.GetSpec()["host"]; ok {
		if dHost, ok := host.(string); ok {
			fqdn := FormatHostnameForPrefixSearch(dHost, n.DestinationRule.GetObjectMeta().Namespace, n.DestinationRule.GetObjectMeta().ClusterName)
			if !n.hasMatchingService(fqdn.Service) {
				validation := models.BuildCheck("This host has no matching workloads", "error", "spec/host")
				validations = append(validations, &validation)
				valid = false
			}
			if subsets, ok := n.DestinationRule.GetSpec()["subsets"]; ok {
				if dSubsets, ok := subsets.([]interface{}); ok {
					// Check that each subset has a matching workload somewhere..
					for i, subset := range dSubsets {
						if innerSubset, ok := subset.(map[string]interface{}); ok {
							if labels, ok := innerSubset["labels"]; ok {
								if dLabels, ok := labels.(map[string]interface{}); ok {
									stringLabels := make(map[string]string, len(dLabels))
									for k, v := range dLabels {
										if s, ok := v.(string); ok {
											stringLabels[k] = s
										}
									}
									if !n.hasMatchingWorkload(fqdn.Service, stringLabels) {
										validation := models.BuildCheck("This subset's labels are not found from any matching host", "error", "spec/subsets["+strconv.Itoa(i)+"]")
										validations = append(validations, &validation)
										valid = false
									}
								}
							}
						}
					}

				}
			}
		}
	}

	return validations, valid
}

func (n NoDestinationChecker) hasMatchingWorkload(service string, labels map[string]string) bool {
	appLabel := config.Get().IstioLabels.AppLabelName
	for _, wl := range n.WorkloadList.Workloads {
		if service == wl.Labels[appLabel] {
			valid := true
			for k, v := range labels {
				wlv, found := wl.Labels[k]
				if !found || wlv != v {
					valid = false
					break
				}
			}
			if valid {
				return true
			}
		}
	}
	return false
}

func (n NoDestinationChecker) hasMatchingService(service string) bool {
	appLabel := config.Get().IstioLabels.AppLabelName
	for _, wl := range n.WorkloadList.Workloads {
		if service == wl.Labels[appLabel] {
			return true
		}
	}
	return false
}
