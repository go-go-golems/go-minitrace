package minitrace

import (
	"slices"
	"time"
)

func ComputeActiveDuration(timestamps []time.Time, threshold time.Duration) *float64 {
	if len(timestamps) < 2 {
		return nil
	}
	if threshold <= 0 {
		threshold = IdleThreshold
	}

	sorted := sortedTimes(timestamps)
	active := 0.0
	for i := 1; i < len(sorted); i++ {
		gap := sorted[i].Sub(sorted[i-1]).Seconds()
		if gap <= threshold.Seconds() {
			active += gap
		}
	}

	return ptr(active)
}

func ComputeTiming(allTimestamps []time.Time) Timing {
	if len(allTimestamps) == 0 {
		return Timing{
			PrivacyLevel:          "full",
			DurationSeconds:       nil,
			ActiveDurationSeconds: nil,
			StartedAt:             nil,
			EndedAt:               nil,
			HourOfDay:             nil,
			DayOfWeek:             nil,
		}
	}

	sorted := sortedTimes(allTimestamps)
	startedAt := sorted[0]
	endedAt := sorted[len(sorted)-1]
	duration := endedAt.Sub(startedAt).Seconds()
	hour := startedAt.Hour()
	day := int(startedAt.Weekday())
	if day == 0 {
		day = 6
	} else {
		day--
	}

	startedAtFormatted := FormatTimestamp(startedAt)
	endedAtFormatted := FormatTimestamp(endedAt)

	return Timing{
		PrivacyLevel:          "full",
		DurationSeconds:       ptr(duration),
		ActiveDurationSeconds: ComputeActiveDuration(sorted, IdleThreshold),
		StartedAt:             &startedAtFormatted,
		EndedAt:               &endedAtFormatted,
		HourOfDay:             &hour,
		DayOfWeek:             &day,
	}
}

func ComputeToolCallContext(toolCalls []ToolCall) {
	totalToolCalls := len(toolCalls)
	if totalToolCalls == 0 {
		return
	}

	for i := range toolCalls {
		position := float64(i) / float64(totalToolCalls)
		toolCalls[i].Context.PositionInSession = &position

		start := max(0, i-5)
		toolsBefore := make([]string, 0, i-start)
		for j := start; j < i; j++ {
			toolsBefore = append(toolsBefore, toolCalls[j].ToolName)
		}
		toolCalls[i].Context.ToolsBefore = toolsBefore
	}
}

func ComputeMetrics(turns []Turn, toolCalls []ToolCall, timing Timing, subagentCount int, tokenTotals *TokenTotals) Metrics {
	operationCounts := map[string]int{}
	for _, toolCall := range toolCalls {
		operationCounts[toolCall.OperationType]++
	}

	toolCallCount := len(toolCalls)
	readCount := operationCounts["READ"]
	metrics := Metrics{
		TurnCount:            len(turns),
		ToolCallCount:        toolCallCount,
		ReadCount:            readCount,
		ModifyCount:          operationCounts["MODIFY"],
		CreateCount:          operationCounts["NEW"],
		ExecuteCount:         operationCounts["EXECUTE"],
		DelegateCount:        operationCounts["DELEGATE"],
		ReadRatio:            nil,
		TimeToFirstAction:    nil,
		IdleRatio:            nil,
		SubagentCount:        subagentCount,
		SubagentToolCalls:    0,
		ModelSwitches:        nil,
		UniqueModels:         nil,
		MedianResponseTokens: nil,
		MaxResponseTokens:    nil,
	}

	if toolCallCount > 0 {
		readRatio := float64(readCount) / float64(toolCallCount)
		readRatio = float64(int(readRatio*1000)) / 1000
		metrics.ReadRatio = &readRatio
	}

	if tokenTotals != nil {
		if tokenTotals.Input != 0 {
			metrics.TotalInputTokens = ptr(tokenTotals.Input)
		}
		if tokenTotals.Output != 0 {
			metrics.TotalOutputTokens = ptr(tokenTotals.Output)
		}
		if tokenTotals.CacheRead != 0 {
			metrics.TotalCacheReadTokens = ptr(tokenTotals.CacheRead)
		}
		if tokenTotals.CacheCreation != 0 {
			metrics.TotalCacheCreationTokens = ptr(tokenTotals.CacheCreation)
		}
		if tokenTotals.Reasoning != 0 {
			metrics.TotalReasoningTokens = ptr(tokenTotals.Reasoning)
		}
		if tokenTotals.Tool != 0 {
			metrics.TotalToolTokens = ptr(tokenTotals.Tool)
		}
		metrics.SessionCost = tokenTotals.Cost
	}

	if timing.StartedAt != nil && toolCallCount > 0 && toolCalls[0].Timestamp != nil {
		startedAt, ok := ParseTimestamp(*timing.StartedAt)
		if ok {
			firstToolCallTime, ok := ParseTimestamp(*toolCalls[0].Timestamp)
			if ok {
				value := firstToolCallTime.Sub(startedAt).Seconds()
				metrics.TimeToFirstAction = &value
			}
		}
	}

	if timing.ActiveDurationSeconds != nil && timing.DurationSeconds != nil && *timing.DurationSeconds > 0 {
		idleRatio := 1 - (*timing.ActiveDurationSeconds / *timing.DurationSeconds)
		idleRatio = float64(int(idleRatio*1000)) / 1000
		metrics.IdleRatio = &idleRatio
	}

	models := make([]string, 0, len(turns))
	responseTokens := make([]int, 0, len(turns))
	for _, turn := range turns {
		if turn.Model != nil && *turn.Model != "" {
			models = append(models, *turn.Model)
		}
		if turn.Role == "assistant" && turn.Usage != nil && turn.Usage.OutputTokens != nil {
			responseTokens = append(responseTokens, *turn.Usage.OutputTokens)
		}
	}

	if len(models) >= 2 {
		unique := map[string]struct{}{}
		switches := 0
		for i, model := range models {
			unique[model] = struct{}{}
			if i > 0 && models[i-1] != model {
				switches++
			}
		}
		metrics.ModelSwitches = ptr(switches)
		metrics.UniqueModels = ptr(len(unique))
	}

	if len(responseTokens) > 0 {
		slices.Sort(responseTokens)
		median := responseTokens[len(responseTokens)/2]
		maximum := responseTokens[len(responseTokens)-1]
		metrics.MedianResponseTokens = ptr(median)
		metrics.MaxResponseTokens = ptr(maximum)
	}

	return metrics
}

func AssignQualityTier(turns []Turn, toolCalls []ToolCall) string {
	hasConversation := len(turns) > 0
	hasToolIO := false
	for _, toolCall := range toolCalls {
		if toolCall.Output.Result != nil {
			hasToolIO = true
			break
		}
	}

	switch {
	case hasConversation && hasToolIO && len(toolCalls) > 10 && len(turns) > 5:
		return "A"
	case hasConversation:
		return "B"
	case !hasConversation:
		return "C"
	default:
		return "D"
	}
}
