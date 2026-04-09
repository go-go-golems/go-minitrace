package serve

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	fields "github.com/go-go-golems/glazed/pkg/cmds/fields"
	apiv1 "github.com/go-go-golems/go-minitrace/gen/proto/go_go_golems/minitrace/api/v1"
	minitracecmd "github.com/go-go-golems/go-minitrace/pkg/minitracecmd"
	queryengine "github.com/go-go-golems/go-minitrace/pkg/query"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/structpb"
)

func (s *Server) handleGetQueryCommandsV2(w http.ResponseWriter, _ *http.Request) {
	catalog, err := minitracecmd.LoadEmbeddedCatalog()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	payload := &apiv1.ListQueryCommandsResponse{
		Meta:     &apiv1.ApiMeta{SchemaVersion: apiSchemaVersion},
		Commands: protoQueryCommands(catalog),
	}
	writeProtoJSON(w, http.StatusOK, payload)
}

func (s *Server) handleExecuteQueryCommandV2(w http.ResponseWriter, r *http.Request) {
	catalog, err := minitracecmd.LoadEmbeddedCatalog()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	routePath := strings.TrimSpace(r.PathValue("path"))
	if !strings.HasSuffix(routePath, "/execute") {
		writeError(w, http.StatusNotFound, "query command execute route not found")
		return
	}
	commandPath := strings.TrimSuffix(routePath, "/execute")
	commandPath = strings.TrimSpace(commandPath)
	command, ok := catalog.ByPath[commandPath]
	if !ok || command == nil {
		writeError(w, http.StatusNotFound, "query command not found")
		return
	}

	var req apiv1.ExecuteQueryCommandRequest
	if err := decodeProtoJSONRequest(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	values := requestValuesToMap(req.GetValues())
	resolvedCommand, resolvedValues, err := minitracecmd.ResolveAliasCommand(catalog, command, values)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, minitracecmd.ErrAliasTargetNotFound) {
			status = http.StatusInternalServerError
		}
		writeError(w, status, err.Error())
		return
	}

	sqlText, err := minitracecmd.RenderCommand(resolvedCommand, minitracecmd.RenderContext{
		TableName: s.tableName,
		Values:    resolvedValues,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	payload := &apiv1.ExecuteQueryCommandResponse{
		Meta:        &apiv1.ApiMeta{SchemaVersion: apiSchemaVersion},
		RenderedSql: sqlText,
		Columns:     []string{},
		Rows:        []*structpb.Struct{},
		DurationMs:  0,
		RowCount:    0,
	}
	if req.GetRenderOnly() {
		writeProtoJSON(w, http.StatusOK, payload)
		return
	}

	if s.conn == nil {
		writeError(w, http.StatusInternalServerError, "query connection is not initialized")
		return
	}
	if err := queryengine.ValidateReadOnlyQuery(sqlText); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	start := time.Now()
	columns, rows, err := executeQueryRows(r.Context(), s.conn, sqlText)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	rowStructs := make([]*structpb.Struct, 0, len(rows))
	for _, row := range rows {
		pbRow, err := structpb.NewStruct(row)
		if err != nil {
			writeError(w, http.StatusInternalServerError, errors.Wrap(err, "encoding query-command row").Error())
			return
		}
		rowStructs = append(rowStructs, pbRow)
	}

	payload.Columns = columns
	payload.Rows = rowStructs
	payload.DurationMs = time.Since(start).Milliseconds()
	payload.RowCount = int32(len(rows))
	writeProtoJSON(w, http.StatusOK, payload)
}

func protoQueryCommands(catalog *minitracecmd.Catalog) []*apiv1.QueryCommand {
	ret := make([]*apiv1.QueryCommand, 0, len(catalog.Commands))
	for _, command := range catalog.Commands {
		ret = append(ret, protoQueryCommand(catalog, command))
	}
	return ret
}

func protoQueryCommand(catalog *minitracecmd.Catalog, command *minitracecmd.MinitraceCommand) *apiv1.QueryCommand {
	displayCommand := command
	flags := command.Flags
	arguments := command.Arguments
	if command.Kind == minitracecmd.MinitraceCommandAlias {
		if target, ok := catalog.ByName[command.AliasFor]; ok && target != nil {
			displayCommand = target
			flags = target.Flags
			arguments = target.Arguments
		}
	}

	shortDescription := command.Short
	if strings.TrimSpace(shortDescription) == "" {
		shortDescription = displayCommand.Short
	}
	longDescription := command.Long
	if strings.TrimSpace(longDescription) == "" {
		longDescription = displayCommand.Long
	}

	return &apiv1.QueryCommand{
		Name:             command.Name,
		Folder:           command.Folder,
		Path:             command.Path,
		ShortDescription: shortDescription,
		LongDescription:  longDescription,
		Flags:            protoQueryCommandParams(flags, false),
		Arguments:        protoQueryCommandParams(arguments, true),
		Tags:             append([]string(nil), displayCommand.Tags...),
		Readonly:         command.Readonly,
		Kind:             protoQueryCommandKind(command.Kind),
		AliasFor:         command.AliasFor,
	}
}

func protoQueryCommandParams(definitions []*fields.Definition, positional bool) []*apiv1.QueryCommandParam {
	ret := make([]*apiv1.QueryCommandParam, 0, len(definitions))
	for _, definition := range definitions {
		if definition == nil {
			continue
		}
		ret = append(ret, protoQueryCommandParam(definition, positional))
	}
	return ret
}

func protoQueryCommandParam(definition *fields.Definition, positional bool) *apiv1.QueryCommandParam {
	defaultJSON := ""
	if definition.Default != nil {
		if payload, err := json.Marshal(*definition.Default); err == nil {
			defaultJSON = string(payload)
		}
	}

	return &apiv1.QueryCommandParam{
		Name:        definition.Name,
		Type:        string(definition.Type),
		Help:        definition.Help,
		Required:    definition.Required,
		DefaultJson: defaultJSON,
		Choices:     append([]string(nil), definition.Choices...),
		Positional:  positional,
		ShortFlag:   definition.ShortFlag,
	}
}

func protoQueryCommandKind(kind minitracecmd.MinitraceCommandKind) apiv1.QueryCommandKind {
	switch kind {
	case minitracecmd.MinitraceCommandVerb:
		return apiv1.QueryCommandKind_QUERY_COMMAND_KIND_VERB
	case minitracecmd.MinitraceCommandAlias:
		return apiv1.QueryCommandKind_QUERY_COMMAND_KIND_ALIAS
	default:
		return apiv1.QueryCommandKind_QUERY_COMMAND_KIND_UNSPECIFIED
	}
}

func requestValuesToMap(values map[string]*structpb.Value) map[string]any {
	ret := make(map[string]any, len(values))
	for key, value := range values {
		if value == nil {
			ret[key] = nil
			continue
		}
		ret[key] = value.AsInterface()
	}
	return ret
}

func executeQueryRows(ctx context.Context, conn *sql.Conn, sqlText string) ([]string, []map[string]any, error) {
	rows, err := conn.QueryContext(ctx, sqlText)
	if err != nil {
		return nil, nil, errors.Wrap(err, "executing query command")
	}
	defer func() { _ = rows.Close() }()

	columns, err := rows.Columns()
	if err != nil {
		return nil, nil, errors.Wrap(err, "reading query-command columns")
	}

	resultRows := make([]map[string]any, 0)
	for rows.Next() {
		values := make([]any, len(columns))
		scanArgs := make([]any, len(columns))
		for i := range scanArgs {
			scanArgs[i] = &values[i]
		}
		if err := rows.Scan(scanArgs...); err != nil {
			return nil, nil, errors.Wrap(err, "scanning query-command row")
		}

		row := make(map[string]any, len(columns))
		for i, column := range columns {
			row[column] = queryengine.NormalizeValue(values[i])
		}
		resultRows = append(resultRows, row)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, errors.Wrap(err, "iterating query-command rows")
	}

	return columns, resultRows, nil
}
