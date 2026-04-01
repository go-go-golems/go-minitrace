import { BrowserRouter, Routes, Route, Navigate } from "react-router";
import { SessionBrowserPage } from "./pages/SessionBrowserPage";
import { TranscriptViewerPage } from "./pages/TranscriptViewerPage";
import { QueryEditorPage } from "./pages/QueryEditorPage";
import { AppLayout } from "./components/AppLayout/AppLayout";

export function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route element={<AppLayout />}>
          <Route path="/" element={<Navigate to="/sessions" replace />} />
          <Route path="/sessions" element={<SessionBrowserPage />} />
          <Route path="/sessions/:sessionId" element={<TranscriptViewerPage />} />
          <Route path="/query" element={<QueryEditorPage />} />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}
