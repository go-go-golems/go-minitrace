import { useParams, useNavigate } from "react-router";
import Box from "@mui/material/Box";
import CircularProgress from "@mui/material/CircularProgress";
import Typography from "@mui/material/Typography";
import { useGetSessionQuery } from "../api/minitrace";
import { TranscriptViewer } from "../components/TranscriptViewer";

export function TranscriptViewerPage() {
  const { sessionId } = useParams<{ sessionId: string }>();
  const navigate = useNavigate();
  // Detail includes the separate unassociated tool ledger; block-only queries
  // intentionally contain only message-associated calls.
  const query = useGetSessionQuery(sessionId ?? "", { skip: !sessionId });
  if (query.isLoading) {
    return <Box sx={{ display: "flex", justifyContent: "center", alignItems: "center", height: "100%" }}><CircularProgress /></Box>;
  }
  if (query.error || !query.data) {
    return <Box sx={{ p: 4 }}><Typography color="error">Session not found: {sessionId}</Typography></Box>;
  }
  return <TranscriptViewer session={query.data} onBack={() => navigate("/sessions")} onQuerySession={(id) => navigate(`/query?session=${id}`)} />;
}
