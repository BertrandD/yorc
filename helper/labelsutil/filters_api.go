package labelsutil

import "novaforge.bull.com/starlings-janus/janus/helper/labelsutil/internal"

// A Warning is an error during the matching process.
// But it doesn't have the standard Go error semantic as it should be reported but not stop any process.
// It basically means that the label set doesn't match but for an unexpected reason.
type Warning error

// Filter defines the Label filter interface
type Filter interface {
	Matches(labels map[string]string) (bool, error)
}

// MatchesAll checks if all of the given filters match a set of labels
//
// Providing no filters is not considered as an error and will return true.
func MatchesAll(labels map[string]string, filters ...Filter) (bool, Warning) {
	for _, filter := range filters {
		m, err := filter.Matches(labels)
		if err != nil || !m {
			return m, err
		}
	}
	return true, nil
}

// CreateFilter creates a Filter from a given input string
func CreateFilter(filter string) (Filter, error) {
	return internal.FilterFromString(filter)
}
