package reconcilation

const (
	ObjectIsSOURCE uint = 0
	ObjectIsCOPY   uint = 1
	ObjectIsNONE   uint = 2
)

// check if the reconciled object is a copy or a source
func SourceOrCopy(Annotations map[string]string) uint {
	if _, ok := Annotations[AnnotationSourceObject]; !ok {
		// since you will need a regex config, to calculate the desired namespaces, this annotation is also
		// usable for checking, whether the item is a source object or not
		if _, ok := Annotations[AnnotationRegexConfig]; ok {
			return ObjectIsSOURCE
		}
	} else { // set the type of the reconciled object, source or copy
		return ObjectIsCOPY
	}
	return ObjectIsNONE
}
