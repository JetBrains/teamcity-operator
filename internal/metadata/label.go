package metadata

type Labels map[string]string

func getDefaultLabelsFromInstanceName(instanceName string) Labels {
	return Labels{
		"app.kubernetes.io/name":      instanceName,
		"app.kubernetes.io/component": "teamcity-server",
		"app.kubernetes.io/part-of":   "teamcity",
	}
}

func GetLabels(instanceName string, instanceLabels map[string]string) Labels {
	labels := getDefaultLabelsFromInstanceName(instanceName)

	for key, value := range instanceLabels {
		_, isLabelPresent := labels[key]
		if !isLabelPresent {
			labels[key] = value
		}
	}

	return labels
}

func GetStatefulSetLabels(instanceName string, nodeName string, nodeRole string, instanceLabels map[string]string) Labels {
	commonLabels := GetLabels(instanceName, instanceLabels)
	nodeLabels := getNodeLabels(nodeName, nodeRole)
	return mergeLabels(commonLabels, nodeLabels)
}

func getNodeLabels(nodeName string, nodeRole string) Labels {
	return Labels{
		"teamcity.jetbrains.com/node-name": nodeName,
		"teamcity.jetbrains.com/role":      nodeRole,
	}
}

func mergeLabels(l1 Labels, l2 Labels) Labels {
	var merged = make(Labels)
	for k, v := range l1 {
		merged[k] = v
	}
	for key, value := range l2 {
		merged[key] = value
	}
	return merged
}
