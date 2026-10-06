package api

import (
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"

	"github.com/klauspost/compress/zstd"
)

const maxIngestBytes = 48 << 20

var zstdDecoder = mustZstdDecoder()

func mustZstdDecoder() *zstd.Decoder {
	// A nil reader gives a DecodeAll-only decoder that starts no goroutines.
	d, err := zstd.NewReader(nil, zstd.WithDecoderMaxMemory(maxIngestBytes))
	if err != nil {
		panic("api: zstd decoder: " + err.Error())
	}
	return d
}

func acceptableIngestContentType(v string) bool {
	mt, _, err := mime.ParseMediaType(v)
	if err != nil {
		return false
	}
	return mt == "application/x-protobuf"
}

func (s *Server) readIngestBody(w http.ResponseWriter, r *http.Request) ([]byte, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxIngestBytes)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			respondError(w, http.StatusRequestEntityTooLarge, "request body too large")
			return nil, false
		}
		s.logger.Printf("ingest: read body: %v", err)
		respondError(w, http.StatusInternalServerError, "failed to read body")
		return nil, false
	}

	switch strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Encoding"))) {
	case "":
		// Uncompressed stays supported for WIRE_COMPRESSION=none.
		return body, true

	case "zstd":
		out, err := zstdDecoder.DecodeAll(body, nil)
		if err != nil {
			if errors.Is(err, zstd.ErrDecoderSizeExceeded) || errors.Is(err, zstd.ErrWindowSizeExceeded) {
				respondError(w, http.StatusRequestEntityTooLarge, "decompressed body too large")
				return nil, false
			}
			respondError(w, http.StatusBadRequest, "invalid zstd body")
			return nil, false
		}
		return out, true

	default:
		respondError(w, http.StatusUnsupportedMediaType, "unsupported Content-Encoding")
		return nil, false
	}
}
