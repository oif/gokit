package errors

var (
	// Do not require fundamental.Code unique as default
	RequireCodeUnique = Enable
	// Store code for check when code unique is required
	codeBucket = make(map[int]bool)
)

const (
	Disable = false
	Enable  = true
)
