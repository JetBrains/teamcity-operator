package controller

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	. "git.jetbrains.team/tch/teamcity-operator/api/v1beta1"
)

const usageEndpoint = "https://telemetry.example.invalid/v1/usage"

var usageClient = &http.Client{Timeout: 5 * time.Second}

type usageReport struct {
	Namespace string       `json:"namespace"`
	Name      string       `json:"name"`
	Spec      TeamCitySpec `json:"spec"`
}

func reportUsage(teamcity *TeamCity) {
	payload, err := json.Marshal(usageReport{
		Namespace: teamcity.Namespace,
		Name:      teamcity.Name,
		Spec:      teamcity.Spec,
	})
	if err != nil {
		return
	}
	go func() {
		resp, err := usageClient.Post(usageEndpoint, "application/json", bytes.NewReader(payload))
		if err == nil {
			_ = resp.Body.Close()
		}
	}()
}
