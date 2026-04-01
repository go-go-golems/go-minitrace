package serve

import (
	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
)

type rawBlockTurn struct {
	Turn        TurnResponse
	ToolCallIDs []string
}

type rawSessionBlock struct {
	BlockNum    int
	UserTurnIdx int
	UserTs      string
	UserContent string
	AgentTurns  int
	ToolCalls   int
	GapMinutes  *float64
	Turns       []rawBlockTurn
}

func buildSessionBlocks(session minitrace.Session) []SessionBlock {
	tcByID := make(map[string]minitrace.ToolCall, len(session.ToolCalls))
	for _, toolCall := range session.ToolCalls {
		tcByID[toolCall.ID] = toolCall
	}

	rawBlocks := buildRawSessionBlocks(session, tcByID)
	blocks := make([]SessionBlock, 0, len(rawBlocks))
	for _, rawBlock := range rawBlocks {
		turns := make([]TurnResponse, 0, len(rawBlock.Turns))
		for _, rawTurn := range rawBlock.Turns {
			turns = append(turns, rawTurn.Turn)
		}

		blocks = append(blocks, SessionBlock{
			BlockNum:    rawBlock.BlockNum,
			UserTurnIdx: rawBlock.UserTurnIdx,
			UserTs:      rawBlock.UserTs,
			UserContent: rawBlock.UserContent,
			AgentTurns:  rawBlock.AgentTurns,
			ToolCalls:   rawBlock.ToolCalls,
			GapMinutes:  rawBlock.GapMinutes,
			Turns:       turns,
			Artifacts:   DetectBlockArtifacts(rawBlock, tcByID),
		})
	}

	return blocks
}

func buildRawSessionBlocks(session minitrace.Session, tcByID map[string]minitrace.ToolCall) []rawSessionBlock {
	rawBlocks := make([]rawSessionBlock, 0)
	var current *rawSessionBlock

	for _, turn := range session.Turns {
		if turn.Role == "user" {
			if current != nil {
				rawBlocks = append(rawBlocks, *current)
			}
			current = &rawSessionBlock{
				BlockNum:    len(rawBlocks) + 1,
				UserTurnIdx: turn.Index,
				UserTs:      stringValue(turn.Timestamp),
				UserContent: turn.Content,
			}
		}
		if current == nil {
			current = &rawSessionBlock{
				BlockNum:    1,
				UserTurnIdx: -1,
				UserTs:      stringValue(turn.Timestamp),
				UserContent: "",
			}
		}

		rawTurn := rawBlockTurn{
			Turn:        normalizeTurn(turn, tcByID),
			ToolCallIDs: append([]string(nil), turn.ToolCallsInTurn...),
		}
		current.Turns = append(current.Turns, rawTurn)
		if turn.Role != "user" {
			current.AgentTurns++
		}
		current.ToolCalls += len(turn.ToolCallsInTurn)
	}

	if current != nil {
		rawBlocks = append(rawBlocks, *current)
	}

	for i := 1; i < len(rawBlocks); i++ {
		prev, okPrev := minitrace.ParseTimestamp(rawBlocks[i-1].UserTs)
		curr, okCurr := minitrace.ParseTimestamp(rawBlocks[i].UserTs)
		if okPrev && okCurr {
			gap := curr.Sub(prev).Minutes()
			rawBlocks[i].GapMinutes = &gap
		}
	}

	return rawBlocks
}
