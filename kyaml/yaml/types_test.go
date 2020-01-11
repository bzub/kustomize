// Copyright 2019 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package yaml

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test that non-UTF8 characters in comments don't cause failures
func TestRNode_GetMeta_UTF16(t *testing.T) {
	sr, err := Parse(`apiVersion: rbac.istio.io/v1alpha1
kind: ServiceRole
metadata:
  name: wildcard
  namespace: default
  # If set to [“*”], it refers to all services in the namespace
  annotations:
    foo: bar
spec:
  rules:
    # There is one service in default namespace, should not result in a validation error
    # If set to [“*”], it refers to all services in the namespace
    - services: ["*"]
      methods: ["GET", "HEAD"]
`)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	actual, err := sr.GetMeta()
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	expected := ResourceMeta{
		APIVersion: "rbac.istio.io/v1alpha1",
		Kind:       "ServiceRole",
		ObjectMeta: ObjectMeta{
			Name:        "wildcard",
			Namespace:   "default",
			Annotations: map[string]string{"foo": "bar"},
		},
	}
	if !assert.Equal(t, expected, actual) {
		t.FailNow()
	}
}

// Test that deepField detects nested fields.
func TestRNode_deepField(t *testing.T) {
	deploy, err := Parse(`apiVersion: apps/v1
kind: Deployment
map:
  name: value
`)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	path, name := deploy.deepField("name")

	expectedPath := []string{"map", "name"}
	if !assert.Equal(t, expectedPath, path) {
		t.FailNow()
	}

	expectedMapN := &MapNode{
		Key:   NewScalarRNode("name"),
		Value: NewScalarRNode("value"),
	}
	if !assert.Equal(t, expectedMapN.Key.YNode().Value, name.Key.YNode().Value) {
		t.FailNow()
	}
	if !assert.Equal(t, expectedMapN.Value.YNode().Value, name.Value.YNode().Value) {
		t.FailNow()
	}
}

// Test that deepField returns empty path, nil when field is not found.
func TestRNode_deepField_NotFound(t *testing.T) {
	deploy, err := Parse(`apiVersion: apps/v1
kind: Deployment
map:
  notName: value
`)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	path, name := deploy.deepField("name")

	if !assert.Len(t, path, 0) {
		t.FailNow()
	}

	if !assert.Nil(t, name) {
		t.FailNow()
	}
}

// Test that deepField does not traverse non-MappingNode(s).
func TestRNode_deepField_NonMappingNodes(t *testing.T) {
	deploy, err := Parse(`apiVersion: apps/v1
kind: Deployment
map:
  list:
  - name: value
`)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	path, name := deploy.deepField("name")

	if !assert.Len(t, path, 0) {
		t.FailNow()
	}

	if !assert.Nil(t, name) {
		t.FailNow()
	}
}
