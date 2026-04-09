export type QueryCommandKind = "verb" | "alias";

export interface QueryCommandParam {
  name: string;
  type: string;
  help: string;
  required: boolean;
  defaultJson: string;
  choices: string[];
  positional: boolean;
  shortFlag: string;
}

export interface QueryCommand {
  name: string;
  folder: string;
  path: string;
  shortDescription: string;
  longDescription: string;
  flags: QueryCommandParam[];
  arguments: QueryCommandParam[];
  tags: string[];
  readonly: boolean;
  kind: QueryCommandKind;
  aliasFor: string;
}
