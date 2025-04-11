package reconcilation

import (
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
)

const (
	LabelEnableMirror      string = "echosec.jnnkrdb.de/mirror-me"
	AnnotationRegexAvoid   string = "echosec.jnnkrdb.de/rgx.avoid"
	AnnotationRegexMatch   string = "echosec.jnnkrdb.de/rgx.match"
	AnnotationSourceObject string = "echosec.jnnkrdb.de/src.object"
	LabelSourceObject      string = AnnotationSourceObject
)

// creating labeselector, to receive all objects, which were cloned
// from the uid's source object
func ObjectsLabelSelector(uid types.UID) labels.Selector {
	var labelSelector map[string]string = make(map[string]string)
	labelSelector[LabelSourceObject] = string(uid)
	return labels.SelectorFromSet(labelSelector)
}
