package tokens

type Estimator interface {
	Estimate(content string) int
	EstimateBytes(content []byte) int
}

type HeuristicEstimator struct{}

func NewHeuristicEstimator() *HeuristicEstimator {
	return &HeuristicEstimator{}
}

func (e *HeuristicEstimator) Estimate(content string) int {
	return len(content) / 3
}

func (e *HeuristicEstimator) EstimateBytes(content []byte) int {
	return len(content) / 3
}
