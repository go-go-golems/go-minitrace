package serve

import (
	"net/http"

	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

var protoJSONMarshaler = protojson.MarshalOptions{
	UseProtoNames: false,
}

func writeProtoJSON(w http.ResponseWriter, status int, msg proto.Message) {
	payload, err := protoJSONMarshaler.Marshal(msg)
	if err != nil {
		log.Error().Err(err).Msg("marshaling protobuf JSON response")
		writeError(w, http.StatusInternalServerError, "failed to marshal protobuf response")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := w.Write(payload); err != nil {
		log.Error().Err(err).Msg("writing protobuf JSON response")
	}
}
