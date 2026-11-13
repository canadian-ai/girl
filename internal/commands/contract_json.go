package commands

import (
	jsonv1 "encoding/json"
	"encoding/json/jsontext"
	"encoding/json/v2"
)

// marshalContractJSON uses the v2 API while retaining the published CLI and GRP
// contracts: null collections, legacy omitempty, HTML escaping, sorted map keys,
// and two-space indentation. Native v2 defaults can be adopted in a versioned
// contract rather than silently changing inputs consumed by agents and CI.
func marshalContractJSON(v any) ([]byte, error) {
	return json.Marshal(v, jsonv1.DefaultOptionsV1(), jsontext.WithIndent("  "))
}
