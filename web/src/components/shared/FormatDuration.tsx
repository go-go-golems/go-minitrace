interface FormatDurationProps {
  seconds: number;
}

/** Human-readable duration: "2.5h", "45m", "12s" */
export function FormatDuration({ seconds }: FormatDurationProps) {
  if (seconds >= 3600) {
    return <span>{(seconds / 3600).toFixed(1)}h</span>;
  }
  if (seconds >= 60) {
    return <span>{Math.round(seconds / 60)}m</span>;
  }
  return <span>{Math.round(seconds)}s</span>;
}

/** Format a duration as "24.7h (3.1h active)" */
export function FormatWallActive({
  wallSeconds,
  activeSeconds,
}: {
  wallSeconds: number;
  activeSeconds: number;
}) {
  return (
    <span>
      <FormatDuration seconds={wallSeconds} />
      <span style={{ opacity: 0.5 }}>
        {" "}
        (<FormatDuration seconds={activeSeconds} /> active)
      </span>
    </span>
  );
}
