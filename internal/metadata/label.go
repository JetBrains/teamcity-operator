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

	for label, value := range instanceLabels {
		_, isLabelPresent := labels[label]
		if !isLabelPresent {
			labels[label] = value
		}
	}

	return labels
}

func GetLabelSelector(instanceName string) Labels {
	return getDefaultLabelsFromInstanceName(instanceName)
}
