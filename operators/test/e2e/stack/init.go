// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package stack

import (
	"fmt"
	"testing"

	estype "github.com/elastic/cloud-on-k8s/operators/pkg/apis/elasticsearch/v1alpha1"
	kbtype "github.com/elastic/cloud-on-k8s/operators/pkg/apis/kibana/v1alpha1"
	"github.com/elastic/cloud-on-k8s/operators/pkg/utils/k8s"
	"github.com/elastic/cloud-on-k8s/operators/test/e2e/helpers"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// InitTestSteps includes pre-requisite tests (eg. is k8s accessible),
// and cleanup from previous tests
func InitTestSteps(stack Builder, k *helpers.K8sHelper) []helpers.TestStep {
	return []helpers.TestStep{

		{
			Name: "K8S should be accessible",
			Test: func(t *testing.T) {
				pods := corev1.PodList{}
				err := k.Client.List(&client.ListOptions{}, &pods)
				require.NoError(t, err)
			},
		},

		{
			Name: "Stack CRDs should exist",
			Test: func(t *testing.T) {
				crds := []runtime.Object{
					&estype.ElasticsearchList{},
					&kbtype.KibanaList{},
				}
				for _, crd := range crds {
					err := k.Client.List(&client.ListOptions{}, crd)
					require.NoError(t, err)
				}
			},
		},

		{
			Name: "Remove the stack if it already exists",
			Test: func(t *testing.T) {
				for _, obj := range stack.RuntimeObjects() {
					err := k.Client.Delete(obj)
					if err != nil {
						// might not exist, which is ok
						require.True(t, apierrors.IsNotFound(err))
					}
				}
				// wait for ES pods to disappear
				helpers.Eventually(func() error {
					return k.CheckPodCount(helpers.ESPodListOptions(stack.Elasticsearch.Name), 0)
				})(t)

				// it may take some extra time for Elasticsearch to be fully deleted
				helpers.Eventually(func() error {
					var es estype.Elasticsearch
					err := k.Client.Get(k8s.ExtractNamespacedName(&stack.Elasticsearch), &es)
					if err != nil && !apierrors.IsNotFound(err) {
						return err
					}
					if err == nil {
						return fmt.Errorf("elasticsearch %s is still there")
					}
					return nil
				})(t)
			},
		},
	}
}
