export interface FocusedTranscriptTarget {
  scopeType: "session" | "turn" | "tool_call";
  targetId: string;
  nonce: number;
}
