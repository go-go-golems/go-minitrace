import { useMemo } from "react";
import { useParams, useNavigate } from "react-router";
import Box from "@mui/material/Box";
import CircularProgress from "@mui/material/CircularProgress";
import Typography from "@mui/material/Typography";
import {
  useGetSessionBlocksQuery,
  useGetSessionSummaryQuery,
} from "../api/minitrace";
import { TranscriptViewer } from "../components/TranscriptViewer";

export function TranscriptViewerPage() {
  const { sessionId } = useParams<{ sessionId: string }>();
  const navigate = useNavigate();

  const summaryQuery = useGetSessionSummaryQuery(sessionId ?? "", {
    skip: !sessionId,
  });
  const blocksQuery = useGetSessionBlocksQuery(sessionId ?? "", {
    skip: !sessionId,
  });

  const session = useMemo(() => {
    if (!summaryQuery.data || !blocksQuery.data) {
      return null;
    }
    return {
      ...summaryQuery.data,
      blocks: blocksQuery.data,
    };
  }, [blocksQuery.data, summaryQuery.data]);

  if (summaryQuery.isLoading || blocksQuery.isLoading) {
    return (
      <Box sx={{ display: "flex", justifyContent: "center", alignItems: "center", height: "100%" }}>
        <CircularProgress />
      </Box>
    );
  }

  if (summaryQuery.error || blocksQuery.error || !session) {
    return (
      <Box sx={{ p: 4 }}>
        <Typography color="error">Session not found: {sessionId}</Typography>
      </Box>
    );
  }

  return (
    <TranscriptViewer
      session={session}
      onBack={() => navigate("/sessions")}
      onQuerySession={(id) => navigate(`/query?session=${id}`)}
    />
  );
}
