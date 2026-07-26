package quality

type categoryDefinition struct {
	id     string
	label  string
	weight int
}

var deterministicCategories = []categoryDefinition{
	{id: "essentials", label: "Essentials and focus", weight: 15},
	{id: "structure", label: "Structure and completeness", weight: 10},
	{id: "evidence", label: "Evidence signals", weight: 15},
	{id: "clarity", label: "Clarity mechanics", weight: 12},
	{id: "consistency", label: "Consistency and chronology", weight: 8},
}

type semanticDefinition struct {
	ruleID string
	label  string
	weight int
}

var semanticDefinitions = []semanticDefinition{
	{ruleID: "semantic.impact_strength", label: "Strength of impact and ownership", weight: 12},
	{ruleID: "semantic.clarity_specificity", label: "Clarity and specificity", weight: 10},
	{ruleID: "semantic.target_relevance", label: "Relevance to the target role", weight: 10},
	{ruleID: "semantic.voice_coherence", label: "Voice and cross-section coherence", weight: 8},
}

func SemanticRuleIDs() []string {
	ids := make([]string, 0, len(semanticDefinitions))
	for _, definition := range semanticDefinitions {
		ids = append(ids, definition.ruleID)
	}
	return ids
}

func semanticDefinitionFor(ruleID string) (semanticDefinition, bool) {
	for _, definition := range semanticDefinitions {
		if definition.ruleID == ruleID {
			return definition, true
		}
	}
	return semanticDefinition{}, false
}
