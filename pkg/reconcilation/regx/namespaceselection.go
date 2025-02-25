package regx

import (
	"regexp"
	"strings"

	"github.com/jnnkrdb/echosec/pkg/reconcilation"
)

/*
This object will be parsed from the sources objects annotations. It contains the information about the actual calculation
of the desired namespaces.

The annotation struct will look something like this:

	metadata:
		annotations:
			echosec.jnnkrdb.de/rgx.avoid: "<regex-1>;<regex-2>;<regex-3>"
			echosec.jnnkrdb.de/rgx.match: "<regex-4>;<regex-5>;<regex-6>"
*/
type AnnotationSet map[string]string

// contains a list of regex
type RegexSet []string

// validate the namespace selector for valid regex'
func (rgxSet AnnotationSet) CalculateCollections() (avoids RegexSet, matches RegexSet) {

	// receive the lists from the annotations
	if tmpAvoids, ok := rgxSet[reconcilation.AnnotationRegexAvoid]; ok {
		avoids = strings.Split(strings.Trim(tmpAvoids, ";"), ";")
	}

	if tmpMatches, ok := rgxSet[reconcilation.AnnotationRegexMatch]; ok {
		matches = strings.Split(strings.Trim(tmpMatches, ";"), ";")
	}

	return
}

// calculates two collections of namespaces, which should weither be avoided
// or provided with the requested object
func (rs RegexSet) Contains(s string) (bool, error) {

	// parse through the list of regexes and compile them, then
	// check for matchings with the given namespace
	for _, regExpression := range rs {

		// if an error occurs, thats most likely, to a not compilable regexp
		// even though they were checked, before creation
		match, err := regexp.MatchString(regExpression, s)
		if err != nil {
			return false, err
		}

		// if the regexpression matches, the function will response with true
		if match {
			return true, nil
		}
	}

	// no regex from the list, does match the given namespace
	return false, nil
}
