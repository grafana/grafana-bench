package cypress

type CypressJsonOutput struct {
	Results struct {
		Tool struct {
			Name string `json:"name"`
		} `json:"tool"`
		Summary Summary `json:"summary"`
		Tests   []Tests `json:"tests"`
	} `json:"results"`
}

type Summary struct {
	Suites  int   `json:"suites"`
	Tests   int   `json:"tests"`
	Failed  int   `json:"failed"`
	Passed  int   `json:"passed"`
	Skipped int   `json:"skipped"`
	Pending int   `json:"pending"`
	Other   int   `json:"other"`
	Start   int64 `json:"start"`
	Stop    int64 `json:"stop"`
}

type Tests struct {
	Name      string `json:"name"`
	Status    string `json:"status"`
	Duration  int    `json:"duration"`
	Message   string `json:"message"`
	Trace     string `json:"trace"`
	RawStatus string `json:"rawStatus"`
	Type      string `json:"type"`
	FilePath  string `json:"filePath"`
	Retry     int    `json:"retry"`
	Flake     bool   `json:"flake"`
	Browser   string `json:"browser"`
}
