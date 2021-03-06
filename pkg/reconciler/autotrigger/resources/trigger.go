/*
Copyright 2019 The Knative Authors

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

package resources

import (
	"encoding/json"
	"fmt"
	"github.com/n3wscott/autotrigger/pkg/reconciler/autotrigger/resources/names"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	eventingv1alpha1 "knative.dev/eventing/pkg/apis/eventing/v1alpha1"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/apis/v1alpha1"
	"knative.dev/pkg/ptr"
)

const (
	filterAnnotation = "trigger.eventing.knative.dev/filter"
)

type brokerFilters map[string]string

func (bf brokerFilters) Broker() string {
	if b, ok := bf["broker"]; ok {
		return b
	}
	return "default"
}

func (bf brokerFilters) Filters() *eventingv1alpha1.TriggerFilterAttributes {
	f := make(eventingv1alpha1.TriggerFilterAttributes, 0)
	for k, v := range bf {
		if k == "broker" {
			continue
		}
		f[k] = v
	}
	return &f
}

// MakeTrigger creates a Trigger from a Service object.
func MakeTriggers(addressable *duckv1.AddressableType) ([]*eventingv1alpha1.Trigger, error) {
	rawFilter, ok := addressable.Annotations[filterAnnotation]
	if !ok {
		return []*eventingv1alpha1.Trigger(nil), nil
	}

	filters := make([]brokerFilters, 0)
	if rawFilter == "" || rawFilter == "[{}]" || rawFilter == "[]" {
		filters = append(filters, brokerFilters{})
	} else if err := json.Unmarshal([]byte(rawFilter), &filters); err != nil {
		return nil, fmt.Errorf("failed to extract auto-trigger from service: %s", err.Error())
	}

	triggers := make([]*eventingv1alpha1.Trigger, 0)

	subscriber := &v1alpha1.Destination{
		Ref: &corev1.ObjectReference{
			APIVersion: addressable.APIVersion,
			Kind:       addressable.Kind,
			Name:       addressable.Name,
		},
	}

	for _, filter := range filters {
		t := &eventingv1alpha1.Trigger{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: names.Trigger(addressable) + "-",
				Namespace:    addressable.Namespace,
				OwnerReferences: []metav1.OwnerReference{{
					APIVersion:         addressable.APIVersion,
					Kind:               addressable.Kind,
					Name:               addressable.Name,
					UID:                addressable.UID,
					BlockOwnerDeletion: ptr.Bool(true),
					Controller:         ptr.Bool(true),
				}},
				Labels: MakeLabels(addressable),
			},
			Spec: eventingv1alpha1.TriggerSpec{
				Broker: filter.Broker(),
				Filter: &eventingv1alpha1.TriggerFilter{
					Attributes: filter.Filters(),
				},
				Subscriber: subscriber,
			},
		}
		triggers = append(triggers, t)
	}

	return triggers, nil
}
