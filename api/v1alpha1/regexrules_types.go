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

import "regexp"

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

	// calculates two collections of namespaces, which should weither be avoided
	// or provided with the requested object
	contains := func(_rgxlist []string, _s string) (bool, error) {
		// parse through the list of regexes and compile them, then
		// check for matchings with the given namespace
		for _, _regExpression := range _rgxlist {

			// if an error occurs, thats most likely, to a not compilable regexp
			// even though they were checked, before creation
			_match, _err := regexp.MatchString(_regExpression, _s)
			if _err != nil {
				return false, _err
			}

			// if the regexpression matches, the function will response with true
			if _match {
				return true, nil
			}
		}

		// no regex from the list, does match the given namespace
		return false, nil
	}

	// check if the namespace appears in the avoids regex list, if an error occurs or
	// the list contains the namespace, then return a false value with the error (error/nil)
	if c, err := contains(nrr.Avoid, namespace); err != nil || c {
		return false, err
	}

	// if no error occured and the avoids list does not contain the namespace, check if
	// the matches list contains the namespace. if so, or an error occurs, then return the
	// given error if any and the true value
	if c, err := contains(nrr.Match, namespace); err != nil || c {
		return true, err
	}

	// if nothing matches and also no error occurs, then return nil error and false as a result
	return false, nil
}
