// Copyright 2021 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package resolve

import (
	"fmt"

	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func createPodNode(containerImages []string, initContainerImages []string) (*yaml.RNode, error) {
	node, err := yaml.FromMap(M{
		"apiVersion": "v1",
		"kind":       "Pod",
		"metadata": M{
			"name": "test-pod",
		},
	})
	if err != nil {
		return nil, err
	}
	for index, image := range containerImages {
		if err := node.PipeE(
			yaml.LookupCreate(yaml.SequenceNode, "spec", "containers"),
			yaml.Append(yaml.NewMapRNode(&map[string]string{
				"name":  fmt.Sprintf("container%d", index),
				"image": image,
			}).YNode()),
		); err != nil {
			return nil, err
		}
	}
	for index, image := range initContainerImages {
		if err := node.PipeE(
			yaml.LookupCreate(yaml.SequenceNode, "spec", "initContainers"),
			yaml.Append(yaml.NewMapRNode(&map[string]string{
				"name":  fmt.Sprintf("initcontainer%d", index),
				"image": image,
			}).YNode()),
		); err != nil {
			return nil, err
		}
	}
	return node, nil
}

func createDeploymentNode(containerImages []string, initContainerImages []string) (*yaml.RNode, error) {
	node, err := yaml.FromMap(M{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata": M{
			"name": "test-deployment",
		},
	})
	if err != nil {
		return nil, err
	}
	for index, image := range containerImages {
		if err := node.PipeE(
			yaml.LookupCreate(yaml.SequenceNode, "spec", "template", "spec", "containers"),
			yaml.Append(yaml.NewMapRNode(&map[string]string{
				"name":  fmt.Sprintf("container%d", index),
				"image": image,
			}).YNode()),
		); err != nil {
			return nil, err
		}
	}
	for index, image := range initContainerImages {
		if err := node.PipeE(
			yaml.LookupCreate(yaml.SequenceNode, "spec", "template", "spec", "initContainers"),
			yaml.Append(yaml.NewMapRNode(&map[string]string{
				"name":  fmt.Sprintf("initcontainer%d", index),
				"image": image,
			}).YNode()),
		); err != nil {
			return nil, err
		}
	}
	return node, nil
}
