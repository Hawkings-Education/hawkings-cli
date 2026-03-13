package cli

func moduleGenerationPayload(researchEnabled bool, researchProvider, researchQuality, researchInstructions string, researchIDs []int, promptCustom string) map[string]any {
	payload := map[string]any{
		"async": true,
		"force": false,
	}

	if len(researchIDs) > 0 {
		payload["research_enabled"] = true
		payload["research_ids"] = researchIDs
	} else if researchEnabled {
		payload["research_enabled"] = true
		if researchProvider != "" {
			payload["research_provider"] = researchProvider
		}
		if researchQuality != "" {
			payload["research_quality"] = researchQuality
		}
		if researchInstructions != "" {
			payload["research_instructions"] = researchInstructions
		}
	} else {
		payload["priority"] = "low"
	}

	if promptCustom != "" {
		payload["course_module_prompt_custom"] = promptCustom
	}

	return payload
}

func sectionGenerationPayload(researchEnabled bool, researchProvider, researchQuality, researchInstructions string, researchIDs []int, promptCustom string) map[string]any {
	payload := map[string]any{
		"async": true,
	}

	if len(researchIDs) > 0 {
		payload["research_enabled"] = true
		payload["research_ids"] = researchIDs
	} else if researchEnabled {
		payload["research_enabled"] = true
		if researchProvider != "" {
			payload["research_provider"] = researchProvider
		}
		if researchQuality != "" {
			payload["research_quality"] = researchQuality
		}
		if researchInstructions != "" {
			payload["research_instructions"] = researchInstructions
		}
	}

	if promptCustom != "" {
		payload["course_module_prompt_custom"] = promptCustom
	}

	return payload
}
