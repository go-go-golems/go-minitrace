import { useParams, useNavigate } from "react-router";
import Box from "@mui/material/Box";
import CircularProgress from "@mui/material/CircularProgress";
import Typography from "@mui/material/Typography";
import { useGetSessionQuery } from "../api/minitrace";
import { TranscriptViewer } from "../components/TranscriptViewer";

export function TranscriptViewerPage() {
  const { sessionId } = useParams<{ sessionId: string }>();
  const navigate = useNavigate();
  const { data: session, isLoading, error } = useGetSessionQuery(sessionId ?? "", {
    skip: !sessionId,
  });

  if (isLoading) {
    return (
      <Box sx={{ display: "flex", justifyContent: "center", alignItems: "center", height: "100%" }}>
        <CircularProgress />
      </Box>
    );
  }

  if (error || !session) {
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
