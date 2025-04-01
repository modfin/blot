package ai

import "github.com/modfin/bellman/models"

type Answer struct {
	Answer          string          `json:"answer,omitempty" json-description:"The answer to the question"`
	ConfidenceScore float32         `json:"confidence_score,omitempty" json-minimum:"0.0" json-maximum:"1.0" json-description:"a confidence score between [0.0, 1.0] that denotes how confident the llm model is in the answer given to the question. This scored is assessed by looking at the RAG retrieved documents and comparing it to the answer"`
	Metadata        models.Metadata `json:"-"`
}
