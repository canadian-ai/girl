package diffstats

type FileStat struct {
	Path         string
	AddedLines   int
	DeletedLines int
	ChangedLines int
	IsBinary     bool
	IsRename     bool
	IsGenerated  bool
	IsLockfile   bool
	OldPath      string
}

type DiffStats struct {
	Files         []FileStat
	TotalAdded    int
	TotalDeleted  int
	TotalChanged  int
	TotalFiles    int
	LargestDelta  int
	Categories    []string
	HasBinary     bool
	HasGenerated  bool
	HasLockfile   bool
	HasRename     bool
}
