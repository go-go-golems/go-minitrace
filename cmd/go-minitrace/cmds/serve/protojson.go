package serve

import (
	"bytes"
	"io"
	"net/http"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

var protoJSONMarshaler = protojson.MarshalOptions{
	UseProtoNames: false,
}

var protoJSONUnmarshaler = protojson.UnmarshalOptions{
	DiscardUnknown: false,
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

func decodeProtoJSONRequest(r *http.Request, dest proto.Message) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return errors.Wrap(err, "reading protobuf JSON body")
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return nil
	}
	if err := protoJSONUnmarshaler.Unmarshal(body, dest); err != nil {
		return errors.Wrap(err, "decoding protobuf JSON body")
	}
	return nil
}
