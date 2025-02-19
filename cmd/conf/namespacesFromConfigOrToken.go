package conf

// calculates the namespaces either from the service config
// or the token file
func GetNamespacesFromConfigOrToken() []string {
	return []string{"default"}
}
