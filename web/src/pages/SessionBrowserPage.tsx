import { useMemo } from "react";
import { useNavigate } from "react-router";
import { useSelector, useDispatch } from "react-redux";
import { useGetAnnotationsQuery, useGetSessionsQuery } from "../api/minitrace";
import { SessionBrowser } from "../components/SessionBrowser";
import type { AnnotationCategory } from "../types";
import type { RootState, AppDispatch } from "../store";
import { setFilterText } from "../store";

export function SessionBrowserPage() {
  const navigate = useNavigate();
  const dispatch = useDispatch<AppDispatch>();
  const { filterText } = useSelector((state: RootState) => state.ui);
  const { data: sessions = [] } = useGetSessionsQuery();
  const { data: annotationData } = useGetAnnotationsQuery();
  const annotations = Array.isArray(annotationData) ? annotationData : [];

  const annotationSummaryBySession = useMemo(() => {
    const summary: Record<string, { count: number; categories: AnnotationCategory[] }> = {};
    for (const ann of annotations) {
      const sessionId = ann.SessionID;
      if (!summary[sessionId]) {
        summary[sessionId] = { count: 0, categories: [] };
      }
      summary[sessionId].count += 1;
      if (!summary[sessionId].categories.includes(ann.Category as AnnotationCategory)) {
        summary[sessionId].categories.push(ann.Category as AnnotationCategory);
      }
    }
    return summary;
  }, [annotations]);

  return (
    <SessionBrowser
      sessions={sessions}
      filterText={filterText}
      onFilterChange={(t) => dispatch(setFilterText(t))}
      onSelectSession={(id) => navigate(`/sessions/${id}`)}
      onQuerySession={(id) =>
        navigate(`/query?session=${id}`)
      }
      annotationSummaryBySession={annotationSummaryBySession}
    />
  );
}
