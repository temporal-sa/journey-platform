package compiler

import (
	"github.com/validated-pattern/journey-platform/internal/domain"
)

// Compiler handles journey compilation, canonicalization, and validation.
type Compiler struct {
	canonicalizer *Canonicalizer
	validator     *Validator
}

// New creates a new Compiler.
func New(opts ...ValidationOption) *Compiler {
	return &Compiler{
		canonicalizer: NewCanonicalizer(),
		validator:     NewValidator(opts...),
	}
}

// Compile performs a basic compilation pass.
func (c *Compiler) Compile(input string) (string, error) {
	return input, nil
}

// CanonicalizeGraph canonicalizes an xyflow presentation graph.
func (c *Compiler) CanonicalizeGraph(graph *XYFlowGraph) (*CanonicalizationResult, error) {
	return c.canonicalizer.Canonicalize(graph)
}

// CanonicalizeDraft canonicalizes a domain.GraphDraft.
func (c *Compiler) CanonicalizeDraft(draft *domain.GraphDraft) (*CanonicalizationResult, error) {
	return c.canonicalizer.CanonicalizeDraft(draft)
}

// ValidateGraph validates an XYFlowGraph presentation graph.
func (c *Compiler) ValidateGraph(graph *XYFlowGraph) (*ValidationResult, error) {
	return c.validator.ValidateXYFlowGraph(graph)
}

// ValidateDraft validates a domain.GraphDraft.
func (c *Compiler) ValidateDraft(draft *domain.GraphDraft) (*ValidationResult, error) {
	return c.validator.ValidateDraft(draft)
}
