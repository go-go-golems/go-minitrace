import { useNavigate } from "react-router";
import { useSelector, useDispatch } from "react-redux";
import { useGetSessionsQuery } from "../api/minitrace";
import { SessionBrowser } from "../components/SessionBrowser";
import type { RootState, AppDispatch } from "../store";
import { setFilterText } from "../store";

export function SessionBrowserPage() {
  const navigate = useNavigate();
  const dispatch = useDispatch<AppDispatch>();
  const { filterText } = useSelector((state: RootState) => state.ui);
  const { data: sessions = [] } = useGetSessionsQuery();

  return (
    <SessionBrowser
      sessions={sessions}
      filterText={filterText}
      onFilterChange={(t) => dispatch(setFilterText(t))}
      onSelectSession={(id) => navigate(`/sessions/${id}`)}
      onQuerySession={(id) =>
        navigate(`/query?session=${id}`)
      }
    />
  );
}
