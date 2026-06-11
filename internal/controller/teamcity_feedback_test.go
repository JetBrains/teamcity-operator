package controller

import (
	"strings"
	"testing"

	. "git.jetbrains.team/tch/teamcity-operator/api/v1beta1"
	"git.jetbrains.team/tch/teamcity-operator/internal/resource"
)

func TestStatefulSetRecreateBlockedErrorUserMessage(t *testing.T) {
	err := newStatefulSetRecreateBlockedError("node", "node", []resource.ImmutableStatefulSetFieldChange{
		{
			Field:   "spec.serviceName",
			Current: "(not set)",
			Desired: "tc-sample-one-svc",
		},
	})

	message := err.UserMessage()
	if message == "" {
		t.Fatal("expected non-empty user message")
	}
	for _, expected := range []string{
		`StatefulSet "node"`,
		`node "node"`,
		"spec.serviceName: current=(not set), desired=tc-sample-one-svc",
		"teamcity.jetbrains.com/allow-sts-recreate=\"true\"",
		"delete and recreate",
	} {
		if !strings.Contains(message, expected) {
			t.Fatalf("expected message to contain %q, got: %s", expected, message)
		}
	}
}

func TestIsStatefulSetBuilder(t *testing.T) {
	instance := &TeamCity{}
	builder := &resource.TeamCityResourceBuilder{Instance: instance}

	if !isStatefulSetBuilder(builder.StatefulSet()) {
		t.Fatal("expected main StatefulSet builder to match")
	}
	if !isStatefulSetBuilder(builder.SecondaryStatefulSet()) {
		t.Fatal("expected secondary StatefulSet builder to match")
	}
	if isStatefulSetBuilder(builder.Service()) {
		t.Fatal("did not expect Service builder to match")
	}
	if isStatefulSetBuilder(builder.Ingress()) {
		t.Fatal("did not expect Ingress builder to match")
	}
}

func TestIsStatefulSetImmutableFieldUpdateError(t *testing.T) {
	err := &StatefulSetRecreateBlockedError{}
	if isStatefulSetImmutableFieldUpdateError(err) {
		t.Fatal("did not expect generic error to match immutable field pattern")
	}

	apiErr := fmtError(`StatefulSet.apps "node" is invalid: spec: Forbidden: updates to statefulset spec for fields other than 'replicas', 'ordinals', 'template' are forbidden`)
	if !isStatefulSetImmutableFieldUpdateError(apiErr) {
		t.Fatal("expected API forbidden error to match immutable field pattern")
	}
}

type fmtError string

func (e fmtError) Error() string { return string(e) }
