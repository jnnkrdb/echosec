package selection

import (
	"strconv"
	"testing"
)

func TestRegexListContains(t *testing.T) {

	type TestRgxContains struct {
		Search        string
		RegexList     []string
		ExpectMatch   bool
		ExpectFailure bool
	}

	// TODO: add more testcases to this function
	var tests = []TestRgxContains{
		{
			Search:        "",
			RegexList:     []string{""},
			ExpectMatch:   true,
			ExpectFailure: false,
		},
	}

	// run the tests
	for testIndex, trc := range tests {

		t.Run(strconv.Itoa(testIndex)+" search: "+trc.Search, func(subT *testing.T) {

			subT.Logf("TestRgxContains: %#v", trc)

			match, err := RegexListContains(trc.Search, trc.RegexList)

			// pretty weird checks, but if it is expected, that the function fails,
			// and the function does in fact not fail, then the test also fails
			if (err != nil) != trc.ExpectFailure {
				subT.Fatalf("TestRgxContains is expected to fail: %t // TextRgxContains failed: %t // error message if any: %v", trc.ExpectFailure, (err != nil), err)
			}

			// if the check did not fail, did the search string match, if expected?
			if match != trc.ExpectMatch {
				subT.Fatalf("TestRgxContains is expected to match: %t // TextRgxContains matched: %t", trc.ExpectMatch, match)
			}
		})
	}
}
