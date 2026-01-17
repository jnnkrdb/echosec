/*
Copyright 2025.

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

import "github.com/jnnkrdb/r8r/pkg/utils/selection"

// RegexRules defines the rules, which should be used to calculate the
// requested namespaces
type NamespaceRegexRules struct {

	// +optional
	Avoid []string `json:"avoid,omitempty"`

	// +optional
	Match []string `json:"match,omitempty"`
}

// calculate wether a secret should exist in the given namespace or not
func (nrr NamespaceRegexRules) ShouldExistInNamespace(namespace string) (bool, error) {

	if c, err := selection.RegexListContains(namespace, nrr.Avoid); err != nil || c {
		return false, err
	}

	return selection.RegexListContains(namespace, nrr.Match)
}
