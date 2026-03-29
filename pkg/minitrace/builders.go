package minitrace

func BuildSessionSkeleton(sessionID, agentFramework, sourceFormat, converterVersion string) Session {
	return Session{
		ID:             sessionID,
		SchemaVersion:  SchemaVersion,
		Profile:        "organic",
		ScenarioID:     nil,
		Quality:        nil,
		Title:          nil,
		Summary:        nil,
		Classification: "internal",
		Provenance: Provenance{
			SourceFormat:      sourceFormat,
			SourcePath:        nil,
			ConvertedAt:       FormatTimestamp(NowUTC()),
			ConverterVersion:  converterVersion,
			OriginalSessionID: ptr(sessionID),
		},
		Flags: Flags{
			ForResearch:   false,
			NeedsCleaning: true,
			ContainsError: false,
			ContainsPII:   false,
			Category:      []string{},
		},
		Environment: Environment{
			Model:          nil,
			ModelVersion:   nil,
			Temperature:    nil,
			ToolsEnabled:   []string{},
			SystemPrompt:   nil,
			AgentFramework: ptr(agentFramework),
			AgentVersion:   nil,
			PlatformType:   nil,
			ProviderHint:   ptr("unknown"),
		},
		OperationalContext: OperationalContext{
			WorkingDirectory: nil,
			GitBranch:        nil,
			GitRef:           nil,
			AutonomyLevel:    nil,
			Sandbox:          nil,
			FrameworkConfig:  nil,
		},
		Timing: Timing{
			PrivacyLevel:          "full",
			DurationSeconds:       nil,
			ActiveDurationSeconds: nil,
			StartedAt:             nil,
			EndedAt:               nil,
			HourOfDay:             nil,
			DayOfWeek:             nil,
		},
		Condition: nil,
		Coordination: Coordination{
			ProjectID:          nil,
			PredecessorSession: nil,
			ConcurrentSessions: nil,
			HumanAttention:     "unknown",
		},
		Handover: Handover{
			Received: nil,
			Produced: nil,
		},
		Turns:       []Turn{},
		ToolCalls:   []ToolCall{},
		Outcome:     nil,
		Annotations: []Annotation{},
		Metrics:     Metrics{},
	}
}

func BuildTurn(index int, timestamp *string, role string, source *string, content string) Turn {
	return Turn{
		Index:             index,
		Timestamp:         timestamp,
		Role:              role,
		Source:            source,
		Model:             nil,
		ContentType:       nil,
		InputChannel:      nil,
		Content:           content,
		FrameworkMetadata: nil,
		ToolCallsInTurn:   []string{},
		Thinking:          nil,
		IntentMarkers:     nil,
		Streaming: Streaming{
			WasStreamed: false,
			StreamLog:   nil,
		},
		Usage: nil,
	}
}

func BuildToolCall(
	toolCallID string,
	turnIndex *int,
	timestamp *string,
	toolName string,
	operationType string,
	filePath *string,
	command *string,
	arguments any,
	success bool,
	result any,
	errorText *string,
	durationMS *int,
	frameworkMetadata any,
	spawnedAgent *SpawnedAgent,
	contentOrigin *string,
	redacted *bool,
) ToolCall {
	truncatedResult, fullBytes, fullHash := TruncateContent(result, TruncateLimit)

	normalizedPath := filePath
	if filePath != nil {
		normalized := NormalizePath(*filePath)
		normalizedPath = &normalized
	}

	return ToolCall{
		ID:                toolCallID,
		EmittingTurnIndex: turnIndex,
		Timestamp:         timestamp,
		ToolName:          toolName,
		OperationType:     operationType,
		Input: ToolCallInput{
			FilePath:  normalizedPath,
			Command:   command,
			Arguments: arguments,
		},
		Output: ToolCallOutput{
			Success:       success,
			Result:        truncatedResult,
			Error:         errorText,
			DurationMS:    durationMS,
			Truncated:     fullBytes != nil,
			FullBytes:     fullBytes,
			FullHash:      fullHash,
			FullReference: nil,
			Redacted:      redacted,
			ContentOrigin: contentOrigin,
		},
		Context: ToolCallContext{
			PositionInSession: nil,
			ToolsBefore:       []string{},
			TimeSinceLastUser: nil,
		},
		FrameworkMetadata: frameworkMetadata,
		SpawnedAgent:      spawnedAgent,
	}
}

func BuildAnnotation(
	annotationID string,
	annotator string,
	scopeType string,
	targetID string,
	category string,
	title string,
	detail string,
	tags []string,
	taxonomyMappings *TaxonomyMappings,
) Annotation {
	if tags == nil {
		tags = []string{}
	}

	mappings := TaxonomyMappings{
		Minitrace: []string{},
		Mast:      []string{},
		Toolemu:   []string{},
	}
	if taxonomyMappings != nil {
		mappings = *taxonomyMappings
	}

	return Annotation{
		ID:               annotationID,
		Timestamp:        FormatTimestamp(NowUTC()),
		Annotator:        annotator,
		Scope:            AnnotationScope{Type: scopeType, TargetID: targetID},
		Content:          AnnotationContent{Category: category, Tags: tags, Title: title, Detail: detail},
		TaxonomyMappings: mappings,
		Classification:   nil,
	}
}
