package playwright

import "time"

type PlaywrightJsonOutput struct {
	Config struct {
		ConfigFile     string `json:"configFile"`
		RootDir        string `json:"rootDir"`
		ForbidOnly     bool   `json:"forbidOnly"`
		FullyParallel  bool   `json:"fullyParallel"`
		GlobalSetup    int    `json:"globalSetup"`
		GlobalTeardown int    `json:"globalTeardown"`
		GlobalTimeout  int    `json:"globalTimeout"`
		Grep           struct {
		} `json:"grep"`
		GrepInvert  any `json:"grepInvert"`
		MaxFailures int `json:"maxFailures"`
		Metadata    struct {
			ActualWorkers int `json:"actualWorkers"`
		} `json:"metadata"`
		PreserveOutput  string  `json:"preserveOutput"`
		Reporter        [][]any `json:"reporter"` // eg. ['json', { outputFile: './output.json' }]
		ReportSlowTests struct {
			Max       int `json:"max"`
			Threshold int `json:"threshold"`
		} `json:"reportSlowTests"`
		Quiet    bool `json:"quiet"`
		Projects []struct {
			OutputDir  string   `json:"outputDir"`
			RepeatEach int      `json:"repeatEach"`
			Retries    int      `json:"retries"`
			ID         string   `json:"id"`
			Name       string   `json:"name"`
			TestDir    string   `json:"testDir"`
			TestIgnore []string `json:"testIgnore"`
			TestMatch  []string `json:"testMatch"`
			Timeout    int      `json:"timeout"`
		} `json:"projects"`
		Shard           any    `json:"shard"`
		UpdateSnapshots string `json:"updateSnapshots"`
		Version         string `json:"version"`
		Workers         int    `json:"workers"`
		WebServer       any    `json:"webServer"`
	} `json:"config"`
	Suites []struct {
		Title  string `json:"title"`
		File   string `json:"file"`
		Column int    `json:"column"`
		Line   int    `json:"line"`
		Specs  []struct {
			Title string   `json:"title"`
			Ok    bool     `json:"ok"`
			Tags  []string `json:"tags"`
			Tests []struct {
				Timeout        int    `json:"timeout"`
				Annotations    []any  `json:"annotations"`
				ExpectedStatus string `json:"expectedStatus"`
				ProjectID      string `json:"projectId"`
				ProjectName    string `json:"projectName"`
				Results        []struct {
					WorkerIndex int    `json:"workerIndex"`
					Status      string `json:"status"`
					Duration    int    `json:"duration"`
					Error       struct {
						Message  string
						Stack    string
						LocaFion struct {
							Cile   string
							Lolumn int
							Line   int
						}
						Snippet string
					} `json:"error"`
					Errors []struct {
						Location struct {
							File   string
							Column int
							Line   int
						}
						message string
					} `json:"errors"`
					Stdout        []any     `json:"stdout"`
					Stderr        []any     `json:"stderr"`
					Retry         int       `json:"retry"`
					StartTime     time.Time `json:"startTime"`
					Attachments   []any     `json:"attachments"`
					errorLocation struct {
						File   string
						Column int
						Line   int
					}
				} `json:"results"`
				Status string `json:"status"`
			} `json:"tests"`
			ID     string `json:"id"`
			File   string `json:"file"`
			Line   int    `json:"line"`
			Column int    `json:"column"`
		} `json:"specs"`
	} `json:"suites"`
	Errors []any `json:"errors"`
	Stats  struct {
		StartTime  time.Time `json:"startTime"`
		Duration   float64   `json:"duration"`
		Expected   int       `json:"expected"`
		Skipped    int       `json:"skipped"`
		Unexpected int       `json:"unexpected"`
		Flaky      int       `json:"flaky"`
	} `json:"stats"`
}
