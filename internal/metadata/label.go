package metadata

import (
	"strings"
)

type label map[string]string

func Label(instanceName string) label {
	return label{
		"app.kubernetes.io/name":      instanceName,
		"app.kubernetes.io/component": "teamcity",
		"app.kubernetes.io/part-of":   "teamcity",
	}
}

func GetLabels(instanceName string, instanceLabels map[string]string) label {
	allLabels := Label(instanceName)

	for label, value := range instanceLabels {
		if !strings.HasPrefix(label, "app.kubernetes.io") {
			allLabels[label] = value
		}
	}

	return allLabels
}

func LabelSelector(instanceName string) label {
	return label{
		"app.kubernetes.io/name": instanceName,
	}
}
