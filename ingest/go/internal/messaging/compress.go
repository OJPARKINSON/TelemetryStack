package messaging

import (
	"fmt"
	"sync"

	"github.com/OJPARKINSON/IRacing-Display/ingest/go/internal/config"
	"github.com/klauspost/compress/zstd"
)

const EncodingZstd = "zstd"

var zstdEncoder = sync.OnceValue(func() *zstd.Encoder {
	enc, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedFastest))
	if err != nil {
		panic("messaging: zstd encoder: " + err.Error())
	}
	return enc
})

func compressBatch(cfg *config.Config, data []byte) (payload []byte, encoding string, err error) {
	if cfg.Compression != EncodingZstd {
		return data, "", nil
	}

	out := zstdEncoder().EncodeAll(data, nil)
	if len(out) == 0 && len(data) > 0 {
		return nil, "", fmt.Errorf("zstd produced an empty frame for %d bytes", len(data))
	}
	return out, EncodingZstd, nil
}
