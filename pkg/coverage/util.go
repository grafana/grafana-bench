package coverage

import "encoding/json"

func deepPrint(value any) string {
	s, _ := json.MarshalIndent(value, "", "\t")
	return string(s)
}
