package regx

import (
	"encoding/json"
	"regexp"

	"github.com/jnnkrdb/echosec/pkg/reconcilation"
)

/*
The selector will be parsed from the sources objects annotations. It contains the information about the actual calculation
of the desired namespaces. Here is an example, made with a configmap:

	apiVersion: v1
	kind: ConfigMap
	metadata:
		annotations:
		  echosec.jnnkrdb.de/rgx.config: |
			  { "avoid": [ "<regex-1>", "<regex-2>" ], "match": [ "<regex-3>", "<regex-4>" ] }
		name: echosec-regex-test
	data:
		testvalue: "empty"
*/
func ShouldExistInNamespace(annotations map[string]string, namespace string) (bool, error) {

	var rgxConf = struct {
		// this list contains all regex strings, to calculate the
		// namespaces, in which an object should not exist
		Avoid []string `json:"avoid"`

		// this list contains all regex string, to calculate the
		// namespaces, in which the object should exist
		Match []string `json:"match"`
	}{}

	// receive the object from the annotations
	if tmpAvoids, ok := annotations[reconcilation.AnnotationRegexConfig]; ok {
		if err := json.Unmarshal([]byte(tmpAvoids), &rgxConf); err != nil {
			return false, err
		}
	}

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
	if c, err := contains(rgxConf.Avoid, namespace); err != nil || c {
		return false, err
	}

	// if no error occured and the avoids list does not contain the namespace, check if
	// the matches list contains the namespace. if so, or an error occurs, then return the
	// given error if any and the true value
	if c, err := contains(rgxConf.Match, namespace); err != nil || c {
		return true, err
	}

	// if nothing matches and also no error occurs, then return nil error and false as a result
	return false, nil
}
