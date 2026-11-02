package learning

// DependencyError carries only technical classifications. Never put text, URLs or
// credentials in these fields. Loggers must still whitelist the field values.
type DependencyError struct {
	Service string
	Kind    string
	Status  int
}

func (e *DependencyError) Error() string { return "learning dependency unavailable" }
func (e *DependencyError) Unwrap() error { return ErrUnavailable }
