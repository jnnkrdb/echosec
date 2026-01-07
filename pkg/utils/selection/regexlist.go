package selection

import "regexp"

// calculates wether a list of regex contains a specific regexpression,
// which matches against the given search-string
func RegexListContains(search string, regexList []string) (bool, error) {

	// parse through the list of regexes and compile them, then
	// check for matchings with the given string
	for _, rx := range regexList {

		// if an error occurs, thats most likely, to a not compilable regexp
		// even though they were checked, before creation
		_match, _err := regexp.MatchString(rx, search)
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
