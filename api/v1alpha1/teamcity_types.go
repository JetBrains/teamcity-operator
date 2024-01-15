/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// TeamCitySpec defines the desired state of TeamCity
type TeamCitySpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Image                  string                        `json:"image"`
	Replicas               *int32                        `json:"replicas"`
	Requests               v1.ResourceList               `json:"requests"` // mandatory, since we rely on it with Xmx setup
	Env                    map[string]string             `json:"env,omitempty"`
	Limits                 v1.ResourceList               `json:"limits,omitempty"`
	PersistentVolumeClaims []CustomPersistentVolumeClaim `json:"persistentVolumeClaims"` //mandatory, we need at least one volume for data persistence

	InitContainers []v1.Container `json:"initContainers,omitempty"`

	// +kubebuilder:default:=95
	XmxPercentage int64 `json:"xmxPercentage"`

	// +kubebuilder:default:={runAsUser: 1000, runAsGroup: 1000, fsGroup: 1000}
	PodSecurityContext v1.PodSecurityContext `json:"podSecurityContext"`

	// +kubebuilder:default:={name: tc-server-port, containerPort: 8111}
	TeamCityServerPort v1.ContainerPort `json:"teamCityServerPort,omitempty"`

	// +kubebuilder:default:={failureThreshold: 3, successThreshold: 1, periodSeconds: 20, initialDelaySeconds: 60, timeoutSeconds: 1}
	LivenessProbeSettings v1.Probe `json:"livenessProbeSettings,omitempty"`
	// +kubebuilder:default:={failureThreshold: 3, successThreshold: 1, periodSeconds: 10, initialDelaySeconds: 60, timeoutSeconds: 1}
	ReadinessProbeSettings v1.Probe `json:"readinessProbeSettings,omitempty"`
	// +kubebuilder:default:={failureThreshold: 15, successThreshold: 1, periodSeconds: 20, initialDelaySeconds: 60, timeoutSeconds: 1}
	StartupProbeSettings v1.Probe `json:"startupProbeSettings,omitempty"`

	// +kubebuilder:default:={path: "/healthCheck/ready", scheme: HTTP, port: 8111}
	ReadinessEndpoint v1.HTTPGetAction `json:"readinessEndpoint,omitempty"`
	// +kubebuilder:default:={path: /healthCheck/healthy, scheme: HTTP, port: 8111}
	HealthEndpoint v1.HTTPGetAction `json:"healthEndpoint,omitempty"`
	// +kubebuilder:default:={}
	DatabaseSecret DatabaseSecret `json:"databaseSecret,omitempty"`
	// +kubebuilder:default:={}
	StartupPropertiesConfig map[string]string `json:"startupPropertiesConfig,omitempty"`
	//+kubebuilder:default:={}
	ServiceList []Service `json:"serviceList,omitempty"`
	//+kubebuilder:default:={}
	IngressList []Ingress `json:"ingressList,omitempty"`
}

type DatabaseSecret struct {
	Secret string `json:"secret,omitempty"`
}

type CustomPersistentVolumeClaim struct {
	Name        string                       `json:"name"`
	VolumeMount v1.VolumeMount               `json:"volumeMount"`
	Spec        v1.PersistentVolumeClaimSpec `json:"spec"`
}

// TeamCityStatus defines the observed state of TeamCity
type TeamCityStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	State   string `json:"state"`
	Message string `json:"message"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// TeamCity is the Schema for the teamcities API
type TeamCity struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TeamCitySpec   `json:"spec,omitempty"`
	Status TeamCityStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// TeamCityList contains a list of TeamCity
type TeamCityList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TeamCity `json:"items"`
}

func (instance *TeamCity) StartUpPropertiesConfigProvided() bool {
	return len(instance.Spec.StartupPropertiesConfig) != 0
}

func (instance *TeamCity) DatabaseSecretProvided() bool {
	return instance.Spec.DatabaseSecret.Secret != ""
}

type Ingress struct {
	Name        string            `json:"name,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	IngressSpec netv1.IngressSpec `json:"spec,omitempty"`
}

type Service struct {
	Name        string            `json:"name,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	ServiceSpec v1.ServiceSpec    `json:"spec,omitempty"`
}

func init() {
	SchemeBuilder.Register(&TeamCity{}, &TeamCityList{})
}
