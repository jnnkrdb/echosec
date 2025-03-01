package regx

import (
	"regexp"
	"strings"

	"github.com/jnnkrdb/echosec/pkg/reconcilation"
)

/*
The selector will be parsed from the sources objects annotations. It contains the information about the actual calculation
of the desired namespaces.

The annotation struct will look something like this:

	metadata:
		annotations:
			echosec.jnnkrdb.de/rgx.avoid: "<regex-1>;<regex-2>;<regex-3>"
			echosec.jnnkrdb.de/rgx.match: "<regex-4>;<regex-5>;<regex-6>"
*/
// should an object exist in the given namespace?
func ShouldExistInNamespace(annotations map[string]string, namespace string) (bool, error) {

	var (
		// this list contains all regex strings, to calculate the
		// namespaces, in which an object should not exist
		avoids []string

		// this list contains all regex string, to calculate the
		// namespaces, in which the object should exist
		matches []string
	)
	// receive the lists from the annotations
	if tmpAvoids, ok := annotations[reconcilation.AnnotationRegexAvoid]; ok {
		avoids = strings.Split(strings.Trim(tmpAvoids, ";"), ";")
	}

	if tmpMatches, ok := annotations[reconcilation.AnnotationRegexMatch]; ok {
		matches = strings.Split(strings.Trim(tmpMatches, ";"), ";")
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
	if c, err := contains(avoids, namespace); err != nil || c {
		return false, err
	}

	// if no error occured and the avoids list does not contain the namespace, check if
	// the matches list contains the namespace. if so, or an error occurs, then return the
	// given error if any and the true value
	if c, err := contains(matches, namespace); err != nil || c {
		return true, err
	}

	// if nothing matches and also no error occurs, then return nil error and false as a result
	return false, nil
}
