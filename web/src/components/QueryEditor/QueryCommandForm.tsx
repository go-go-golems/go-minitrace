import Box from "@mui/material/Box";
import Checkbox from "@mui/material/Checkbox";
import FormControlLabel from "@mui/material/FormControlLabel";
import MenuItem from "@mui/material/MenuItem";
import Stack from "@mui/material/Stack";
import TextField from "@mui/material/TextField";
import Typography from "@mui/material/Typography";
import type { QueryCommand, QueryCommandParam } from "../../types";

interface QueryCommandFormProps {
  command: QueryCommand;
  values: Record<string, unknown>;
  onChange: (name: string, value: unknown) => void;
}

export function QueryCommandForm({ command, values, onChange }: QueryCommandFormProps) {
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
