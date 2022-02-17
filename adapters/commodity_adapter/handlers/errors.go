package handlers

// InvalidParam defines the error returned by an invalid parameter.
type InvalidParam struct {
	// Param is the name of the parameter.
	Param string `json:"Parameter"`
	// Value is the value of the Parameter.
	Value string `json:"Value"`
	// Error is the returned error.
	Error error `json:"Error"`
}
