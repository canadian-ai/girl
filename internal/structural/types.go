package structural

type BucketLineCounts struct {
	Logic            int
	Test             int
	ReusableSupport  int
	EphemeralSupport int
	ConfigData       int
	ConfigStructural int
	Generated        int
}

type StructuralRatios struct {
	StructuralOverhead float64
	TestToLogic        float64
	ProductiveScaffold float64
}

type CohesionResult struct {
	Variance         float64
	SuggestedClusters [][]string
}

type Classification struct {
	Added    BucketLineCounts
	Ratios   StructuralRatios
	Cohesion CohesionResult
}
