package sort

import (
	"sort"

	"github.com/grafana/grafana-bench/pkg/executor"
)

func SortTestRunByFilename(tr []executor.TestRunSummary) {
	sort.Slice(tr, func(i, j int) bool {
		return tr[i].TestFile < tr[j].TestFile
	})
}