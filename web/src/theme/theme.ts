import { createTheme } from "@mui/material/styles";

/**
 * Minitrace Explorer theme.
 *
 * Aesthetic direction: "data-observatory" — dark, information-dense,
 * monospace-accented, with warm amber highlights on a cool slate base.
 * Think mission control, not marketing site.
 */
export const theme = createTheme({
  palette: {
    mode: "dark",
    primary: {
      main: "#f5a623",
      light: "#ffc966",
      dark: "#c17a00",
    },
    secondary: {
      main: "#64b5f6",
      light: "#90caf9",
      dark: "#1976d2",
    },
    background: {
      default: "#0d1117",
      paper: "#161b22",
    },
    text: {
      primary: "#e6edf3",
      secondary: "#8b949e",
    },
    success: { main: "#3fb950" },
    error: { main: "#f85149" },
    warning: { main: "#d29922" },
    info: { main: "#58a6ff" },
    divider: "#30363d",
  },
  typography: {
    fontFamily: '"IBM Plex Sans", "Roboto", "Helvetica", "Arial", sans-serif',
    h1: { fontWeight: 600, fontSize: "1.75rem", letterSpacing: "-0.01em" },
    h2: { fontWeight: 600, fontSize: "1.4rem", letterSpacing: "-0.01em" },
    h3: { fontWeight: 600, fontSize: "1.15rem" },
    h4: { fontWeight: 600, fontSize: "1rem" },
    body1: { fontSize: "0.875rem", lineHeight: 1.6 },
    body2: { fontSize: "0.8125rem", lineHeight: 1.5 },
    caption: { fontSize: "0.75rem", letterSpacing: "0.02em" },
    overline: {
      fontSize: "0.6875rem",
      fontWeight: 600,
      letterSpacing: "0.08em",
      textTransform: "uppercase",
    },
  },
  shape: {
    borderRadius: 6,
  },
  components: {
    MuiCssBaseline: {
      styleOverrides: {
        body: {
          scrollbarColor: "#30363d #0d1117",
          "&::-webkit-scrollbar": { width: 8 },
          "&::-webkit-scrollbar-track": { background: "#0d1117" },
          "&::-webkit-scrollbar-thumb": {
            background: "#30363d",
            borderRadius: 4,
          },
        },
      },
    },
    MuiPaper: {
      defaultProps: { elevation: 0 },
      styleOverrides: {
        root: { backgroundImage: "none", border: "1px solid #30363d" },
      },
    },
    MuiChip: {
      styleOverrides: {
        root: { fontWeight: 500, fontSize: "0.75rem" },
      },
    },
    MuiTableCell: {
      styleOverrides: {
        root: {
          borderBottom: "1px solid #21262d",
          fontSize: "0.8125rem",
          padding: "8px 12px",
        },
        head: {
          fontWeight: 600,
          color: "#8b949e",
          textTransform: "uppercase",
          fontSize: "0.6875rem",
          letterSpacing: "0.06em",
        },
      },
    },
    MuiButton: {
      styleOverrides: {
        root: { textTransform: "none", fontWeight: 600 },
      },
    },
  },
});
