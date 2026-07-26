package ai

const validAssessmentsJSON = `[
	{"rule_id":"semantic.impact_strength","verdict":"pass","evidence":"Builds reliable APIs","confidence":0.9,"reason":"Impact is clear."},
	{"rule_id":"semantic.clarity_specificity","verdict":"pass","evidence":"Builds reliable APIs","confidence":0.9,"reason":"The wording is specific."},
	{"rule_id":"semantic.target_relevance","verdict":"not_applicable","evidence":"","confidence":0.9,"reason":"No target role was supplied."},
	{"rule_id":"semantic.voice_coherence","verdict":"pass","evidence":"Builds reliable APIs","confidence":0.9,"reason":"The voice is coherent."}
]`

const validReviewJSON = `{"summary":"Strong base","assessments":` + validAssessmentsJSON + `,"suggestions":[],"warnings":[]}`
