package recipes

type Thresholds struct {
	LargeComponentLines  int
	RepeatedJSXCount     int
	MaxHooksPerComponent int
	MaxStateVars         int
	MaxEffects           int
}

func DefaultThresholds() *Thresholds {
	return &Thresholds{
		LargeComponentLines:  200,
		RepeatedJSXCount:     3,
		MaxHooksPerComponent: 5,
		MaxStateVars:         4,
		MaxEffects:           2,
	}
}
