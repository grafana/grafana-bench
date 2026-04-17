package analyzer

import (
	"sort"
	"time"
)

// RuleVersion is emitted on every ConfirmedDefect event so that the rule can
// be tuned without losing the ability to retroactively segment past data.
const RuleVersion = "v1"

// Confidence levels for a ConfirmedDefect.
const (
	ConfidenceConfirmed = "confirmed"
	ConfidenceSuspected = "suspected"
)

// RuleConfig tunes the detection rule. Zero values fall back to the v1
// defaults inside Analyze().
type RuleConfig struct {
	// MinFailures is the persistence threshold: how many consecutive failing
	// runs on a (service, runStage, testFile, grafanaVersion) are required
	// before we'll consider the failure persistent.
	MinFailures int

	// MinPriorPassing is the regression-boundary threshold: how many passing
	// runs are required on the most recent prior distinct grafanaVersion for
	// us to treat it as "known-good".
	MinPriorPassing int
}

func (r RuleConfig) withDefaults() RuleConfig {
	if r.MinFailures <= 0 {
		r.MinFailures = 3
	}
	if r.MinPriorPassing <= 0 {
		r.MinPriorPassing = 2
	}
	return r
}

// ConfirmedDefect is the output of the rule engine — one per confirmed or
// suspected defect. Emitters translate these into the msg=defectConfirmed
// logfmt schema.
type ConfirmedDefect struct {
	Service              string
	RunStage             string
	TestFile             string
	TestFolder           string
	GrafanaVersion       string
	PriorPassingVersion  string
	GrafanaSlug          string
	GrafanaURL           string
	SignatureHash        string
	Confidence           string
	ConfidenceRuns       int
	PriorPassingRuns     int
	ExitMessageCanonical string
	ExitMessageSample    string
	FirstFailureTime     time.Time
	LastFailureTime      time.Time
	SourceRunIDs         []string
	AnalyzeWindow        time.Duration
	AnalyzedAt           time.Time
	RuleVersion          string
}

// Analyze applies the v1 defect-confirmation rule to a set of testRun records
// and returns one ConfirmedDefect per detection. The returned slice is sorted
// by (service, runStage, testFile, grafanaVersion) for deterministic output.
func Analyze(records []TestRunRecord, cfg RuleConfig, window time.Duration, now time.Time) []ConfirmedDefect {
	cfg = cfg.withDefaults()

	// Drop any non-testRun records the caller might have pulled in by mistake.
	filtered := records[:0:0]
	for _, r := range records {
		if r.Msg == "" || r.Msg == "testRun" {
			filtered = append(filtered, r)
		}
	}

	// Group by (service, runStage, testFile). grafanaVersion is NOT part of
	// the group key here — we need to see all versions for a test so we can
	// locate the regression boundary.
	type perTestKey struct {
		Service  string
		RunStage string
		TestFile string
	}
	groups := map[perTestKey][]TestRunRecord{}
	for _, r := range filtered {
		if r.TestFile == "" || r.Service == "" {
			continue
		}
		k := perTestKey{Service: r.Service, RunStage: r.RunStage, TestFile: r.TestFile}
		groups[k] = append(groups[k], r)
	}

	var defects []ConfirmedDefect
	for _, recs := range groups {
		sort.Slice(recs, func(i, j int) bool { return recs[i].Time.Before(recs[j].Time) })
		defects = append(defects, evaluateTest(recs, cfg, window, now)...)
	}

	sort.Slice(defects, func(i, j int) bool {
		a, b := defects[i], defects[j]
		if a.Service != b.Service {
			return a.Service < b.Service
		}
		if a.RunStage != b.RunStage {
			return a.RunStage < b.RunStage
		}
		if a.TestFile != b.TestFile {
			return a.TestFile < b.TestFile
		}
		return a.GrafanaVersion < b.GrafanaVersion
	})

	return defects
}

// evaluateTest runs the three-part rule for a single (service, runStage,
// testFile) group, returning a defect per grafanaVersion that breaches it.
func evaluateTest(recs []TestRunRecord, cfg RuleConfig, window time.Duration, now time.Time) []ConfirmedDefect {
	// Collapse runs per grafanaVersion in time order, so we can identify the
	// "most recent prior distinct version" deterministically.
	type versionBlock struct {
		Version string
		Records []TestRunRecord
	}
	var blocks []versionBlock
	for _, r := range recs {
		if r.GrafanaVersion == "" {
			continue
		}
		if n := len(blocks); n > 0 && blocks[n-1].Version == r.GrafanaVersion {
			blocks[n-1].Records = append(blocks[n-1].Records, r)
			continue
		}
		blocks = append(blocks, versionBlock{Version: r.GrafanaVersion, Records: []TestRunRecord{r}})
	}

	// Aggregate per-version stats separately for the regression-boundary
	// lookup (it wants to know, for a given prior version, did it have enough
	// passing runs anywhere in the window — not just in the last contiguous
	// block).
	type versionStats struct {
		FirstSeen time.Time
		LastSeen  time.Time
		Passed    int
		Failed    int
		Records   []TestRunRecord
	}
	stats := map[string]*versionStats{}
	for _, r := range recs {
		if r.GrafanaVersion == "" {
			continue
		}
		s := stats[r.GrafanaVersion]
		if s == nil {
			s = &versionStats{FirstSeen: r.Time, LastSeen: r.Time}
			stats[r.GrafanaVersion] = s
		}
		if r.Time.Before(s.FirstSeen) {
			s.FirstSeen = r.Time
		}
		if r.Time.After(s.LastSeen) {
			s.LastSeen = r.Time
		}
		switch {
		case r.Passed():
			s.Passed++
		case r.Failed():
			s.Failed++
		}
		s.Records = append(s.Records, r)
	}

	var defects []ConfirmedDefect
	seenBadVersions := map[string]bool{}

	for i, blk := range blocks {
		// Need at least MinFailures in this block to clear rule (1).
		if len(blk.Records) < cfg.MinFailures {
			continue
		}
		if seenBadVersions[blk.Version] {
			continue
		}

		// Rule (1): persistence — last MinFailures records must all be
		// failures (on the tail of the block, ensuring we're describing a
		// currently-failing state, not a historical blip).
		tail := blk.Records[len(blk.Records)-cfg.MinFailures:]
		allFail := true
		for _, r := range tail {
			if !r.Failed() {
				allFail = false
				break
			}
		}
		if !allFail {
			continue
		}

		// Rule (2): regression boundary — walk back through prior version
		// blocks to find the most recent distinct version with ≥ MinPriorPassing
		// passing runs and 0 failures in its stats.
		priorVersion := ""
		priorPassing := 0
		for j := i - 1; j >= 0; j-- {
			cand := blocks[j].Version
			if cand == blk.Version {
				continue
			}
			st := stats[cand]
			if st == nil {
				continue
			}
			if st.Failed == 0 && st.Passed >= cfg.MinPriorPassing {
				priorVersion = cand
				priorPassing = st.Passed
				break
			}
			// A prior version with any failures disqualifies it — we need a
			// clean green to claim "regressed".
			if st.Failed > 0 {
				break
			}
		}
		if priorVersion == "" {
			continue
		}

		// Rule (3): signature stability across the failing tail.
		var sigs []string
		var canonicalSample string
		var rawSample string
		for _, r := range tail {
			canonical := Canonicalize(r.ExitMessage)
			sig := Signature(r.TestFile, canonical)
			sigs = append(sigs, sig)
			if canonicalSample == "" {
				canonicalSample = canonical
				rawSample = r.ExitMessage
			}
		}
		confidence := ConfidenceConfirmed
		for _, s := range sigs {
			if s != sigs[0] {
				confidence = ConfidenceSuspected
				break
			}
		}

		sourceIDs := make([]string, 0, len(tail))
		for _, r := range tail {
			sourceIDs = append(sourceIDs, r.RunID)
		}

		defects = append(defects, ConfirmedDefect{
			Service:              blk.Records[0].Service,
			RunStage:             blk.Records[0].RunStage,
			TestFile:             blk.Records[0].TestFile,
			TestFolder:           blk.Records[0].Folder,
			GrafanaVersion:       blk.Version,
			PriorPassingVersion:  priorVersion,
			GrafanaSlug:          tail[len(tail)-1].GrafanaSlug,
			GrafanaURL:           tail[len(tail)-1].GrafanaURL,
			SignatureHash:        sigs[0],
			Confidence:           confidence,
			ConfidenceRuns:       len(tail),
			PriorPassingRuns:     priorPassing,
			ExitMessageCanonical: truncate(canonicalSample, 500),
			ExitMessageSample:    truncate(rawSample, 200),
			FirstFailureTime:     tail[0].Time,
			LastFailureTime:      tail[len(tail)-1].Time,
			SourceRunIDs:         sourceIDs,
			AnalyzeWindow:        window,
			AnalyzedAt:           now,
			RuleVersion:          RuleVersion,
		})
		seenBadVersions[blk.Version] = true
	}

	return defects
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
