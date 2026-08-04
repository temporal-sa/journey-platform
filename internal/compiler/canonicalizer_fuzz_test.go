package compiler_test

import (
	"encoding/json"
	"testing"

	"github.com/validated-pattern/journey-platform/internal/compiler"
	"github.com/validated-pattern/journey-platform/internal/domain"
)

// FuzzCanonicalJSON fuzz tests raw JSON unmarshaling and canonicalization of presentation graphs and domain drafts.
func FuzzCanonicalJSON(f *testing.F) {
	seeds := [][]byte{
		[]byte(`{"draft_id":"d1","tenant_id":"t1","name":"Flow","version":1,"nodes":[{"id":"n1","type":"trigger","name":"Start"}],"edges":[]}`),
		[]byte(`{"draft_id":"d2","tenant_id":"t2","name":"Complex","version":2,"viewport":{"x":10.5,"y":20.0,"zoom":1.0},"nodes":[{"id":"n1","type":"action","position":{"x":100,"y":200},"metrics":["m1","m2"]}],"edges":[{"id":"e1","source":"n1","target":"n2","condition":"event.val > 10"}]}`),
		[]byte(`{invalid json payload`),
		[]byte(`{"nodes":null,"edges":null}`),
		[]byte(`{"nodes":[{"id":"","type":""}]}`),
		[]byte("\x00\xff\xfe\xfd\x12\x34"),
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input []byte) {
		c := compiler.New()

		// 1. Test XYFlowGraph unmarshaling + Canonicalize
		var xyGraph compiler.XYFlowGraph
		if err := json.Unmarshal(input, &xyGraph); err == nil {
			_, _ = c.CanonicalizeGraph(&xyGraph)
		}

		// 2. Test GraphDraft unmarshaling + CanonicalizeDraft
		var draft domain.GraphDraft
		if err := json.Unmarshal(input, &draft); err == nil {
			canonicalizer := compiler.NewCanonicalizer()
			_, _ = canonicalizer.CanonicalizeDraft(&draft)
		}
	})
}
