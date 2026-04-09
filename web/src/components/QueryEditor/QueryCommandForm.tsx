import { useState } from "react";
import Accordion from "@mui/material/Accordion";
import AccordionDetails from "@mui/material/AccordionDetails";
import AccordionSummary from "@mui/material/AccordionSummary";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Checkbox from "@mui/material/Checkbox";
import FormControlLabel from "@mui/material/FormControlLabel";
import MenuItem from "@mui/material/MenuItem";
import Stack from "@mui/material/Stack";
import TextField from "@mui/material/TextField";
import Typography from "@mui/material/Typography";
import ExpandMoreIcon from "@mui/icons-material/ExpandMore";
import type { QueryCommand, QueryCommandParam } from "../../types";
import { SqlCodeViewer } from "./SqlCodeViewer";

interface QueryCommandFormProps {
  command: QueryCommand;
  values: Record<string, unknown>;
  renderedSql?: string | null;
  onChange: (name: string, value: unknown) => void;
}

export function QueryCommandForm({ command, values, renderedSql = null, onChange }: QueryCommandFormProps) {
  return (
    <Stack spacing={2} sx={{ height: "100%", overflow: "auto", pr: 1 }}>
      <Box>
        <Typography variant="subtitle1">{command.name}</Typography>
        <Typography variant="body2" color="text.secondary">
          {command.shortDescription || command.longDescription || "No description available."}
        </Typography>
      </Box>

      {command.arguments.length > 0 && (
        <ParameterSection
          title="Arguments"
          params={command.arguments}
          values={values}
          onChange={onChange}
        />
      )}

      {command.flags.length > 0 && (
        <ParameterSection
          title="Flags"
          params={command.flags}
          values={values}
          onChange={onChange}
        />
      )}

      <Stack spacing={1}>
        <Typography variant="overline">Debug helpers</Typography>
        <SqlDebugAccordion
          title="Raw command SQL"
          subtitle={buildRawSqlSubtitle(command)}
          sql={command.rawSql}
          defaultExpanded
          emptyMessage="This command does not expose a raw SQL template."
        />
        <SqlDebugAccordion
          title="Rendered SQL"
          subtitle="Last rendered SQL from a successful command run."
          sql={renderedSql ?? ""}
          emptyMessage="Run the command to populate the rendered SQL preview."
        />
      </Stack>
    </Stack>
  );
}

function ParameterSection({
  title,
  params,
  values,
  onChange,
}: {
  title: string;
  params: QueryCommandParam[];
  values: Record<string, unknown>;
  onChange: (name: string, value: unknown) => void;
}) {
  return (
    <Stack spacing={1.5}>
      <Typography variant="overline">{title}</Typography>
      {params.map((param) => (
        <ParameterField
          key={param.name}
          param={param}
          value={values[param.name]}
          onChange={(value) => onChange(param.name, value)}
        />
      ))}
    </Stack>
  );
}

function ParameterField({
  param,
  value,
  onChange,
}: {
  param: QueryCommandParam;
  value: unknown;
  onChange: (value: unknown) => void;
}) {
  const label = param.shortFlag ? `${param.name} (-${param.shortFlag})` : param.name;
  const helperText = param.help || (param.required ? "Required" : "Optional");

  switch (param.type) {
    case "bool":
      return (
        <FormControlLabel
          control={
            <Checkbox
              checked={Boolean(value)}
              onChange={(event) => onChange(event.target.checked)}
            />
          }
          label={label}
        />
      );

    case "choice":
      return (
        <TextField
          select
          fullWidth
          size="small"
          label={label}
          helperText={helperText}
          value={typeof value === "string" ? value : ""}
          onChange={(event) => onChange(event.target.value)}
        >
          {param.choices.map((choice) => (
            <MenuItem key={choice} value={choice}>
              {choice}
            </MenuItem>
          ))}
        </TextField>
      );

    case "int":
      return (
        <TextField
          fullWidth
          size="small"
          type="number"
          label={label}
          helperText={helperText}
          value={typeof value === "number" ? value : ""}
          onChange={(event) => onChange(event.target.value === "" ? "" : Number(event.target.value))}
        />
      );

    case "float":
      return (
        <TextField
          fullWidth
          size="small"
          type="number"
          label={label}
          helperText={helperText}
          value={typeof value === "number" ? value : ""}
          inputProps={{ step: "any" }}
          onChange={(event) => onChange(event.target.value === "" ? "" : Number(event.target.value))}
        />
      );

    case "date":
      return (
        <TextField
          fullWidth
          size="small"
          type="date"
          label={label}
          helperText={helperText}
          value={typeof value === "string" ? value : ""}
          InputLabelProps={{ shrink: true }}
          onChange={(event) => onChange(event.target.value)}
        />
      );

    case "stringList":
      return (
        <TextField
          fullWidth
          size="small"
          label={label}
          helperText={`${helperText} Use commas to separate multiple values.`}
          value={Array.isArray(value) ? value.join(", ") : ""}
          onChange={(event) => onChange(parseStringList(event.target.value))}
        />
      );

    case "intList":
      return (
        <TextField
          fullWidth
          size="small"
          label={label}
          helperText={`${helperText} Use commas to separate multiple values.`}
          value={Array.isArray(value) ? value.join(", ") : ""}
          onChange={(event) => onChange(parseIntList(event.target.value))}
        />
      );

    case "floatList":
      return (
        <TextField
          fullWidth
          size="small"
          label={label}
          helperText={`${helperText} Use commas to separate multiple values.`}
          value={Array.isArray(value) ? value.join(", ") : ""}
          onChange={(event) => onChange(parseFloatList(event.target.value))}
        />
      );

    case "choiceList":
      return (
        <TextField
          select
          fullWidth
          size="small"
          label={label}
          helperText={helperText}
          value={Array.isArray(value) ? value : []}
          SelectProps={{
            multiple: true,
            renderValue: (selected) => Array.isArray(selected) ? selected.join(", ") : "",
          }}
          onChange={(event) => {
            const nextValue = event.target.value;
            onChange(Array.isArray(nextValue) ? nextValue : [String(nextValue)]);
          }}
        >
          {param.choices.map((choice) => (
            <MenuItem key={choice} value={choice}>
              {choice}
            </MenuItem>
          ))}
        </TextField>
      );

    case "string":
    default:
      return (
        <TextField
          fullWidth
          size="small"
          label={label}
          helperText={helperText}
          value={typeof value === "string" ? value : ""}
          onChange={(event) => onChange(event.target.value)}
        />
      );
  }
}

function buildRawSqlSubtitle(command: QueryCommand): string {
  if (command.kind === "alias" && command.rawSqlPath && command.rawSqlPath !== command.path) {
    return `Alias for ${command.aliasFor || "target command"}. Showing template from ${command.rawSqlPath}.`;
  }
  if (command.rawSqlPath) {
    return `Source template: ${command.rawSqlPath}`;
  }
  return "Underlying sqleton SQL template.";
}

function SqlDebugAccordion({
  title,
  subtitle,
  sql,
  emptyMessage,
  defaultExpanded = false,
}: {
  title: string;
  subtitle: string;
  sql: string;
  emptyMessage: string;
  defaultExpanded?: boolean;
}) {
  const [copyStatus, setCopyStatus] = useState<"idle" | "copied" | "failed">("idle");

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(sql);
      setCopyStatus("copied");
      window.setTimeout(() => setCopyStatus("idle"), 2000);
    } catch {
      setCopyStatus("failed");
      window.setTimeout(() => setCopyStatus("idle"), 2000);
    }
  };

  return (
    <Accordion disableGutters defaultExpanded={defaultExpanded}>
      <AccordionSummary expandIcon={<ExpandMoreIcon />}>
        <Stack spacing={0.25}>
          <Typography variant="subtitle2">{title}</Typography>
          <Typography variant="caption" color="text.secondary">
            {subtitle}
          </Typography>
        </Stack>
      </AccordionSummary>
      <AccordionDetails>
        {sql.trim() ? (
          <Stack spacing={1}>
            <Stack direction="row" justifyContent="flex-end" alignItems="center" spacing={1}>
              {copyStatus === "copied" && (
                <Typography variant="caption" color="success.main">
                  Copied
                </Typography>
              )}
              {copyStatus === "failed" && (
                <Typography variant="caption" color="error.main">
                  Copy failed
                </Typography>
              )}
              <Button size="small" variant="outlined" onClick={handleCopy}>
                Copy SQL
              </Button>
            </Stack>
            <SqlCodeViewer value={sql} minHeight={140} />
          </Stack>
        ) : (
          <Typography variant="body2" color="text.secondary">
            {emptyMessage}
          </Typography>
        )}
      </AccordionDetails>
    </Accordion>
  );
}

function parseStringList(value: string): string[] {
  return value
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
}

function parseIntList(value: string): number[] {
  return value
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean)
    .map((item) => Number(item))
    .filter((item) => !Number.isNaN(item));
}

function parseFloatList(value: string): number[] {
  return value
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean)
    .map((item) => Number(item))
    .filter((item) => !Number.isNaN(item));
}
