package gphotos

import (
	"bytes"
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
)

// Scotty finalize tokens: the opaque upload-finalization blob returned by the
// resumable uploader, validated before it is embedded into the commit request.

const scottyFinalizeTokenVersion = 2

type ScottyFinalizeToken struct {
	Raw []byte
}

func ParseScottyFinalizeToken(raw []byte) (ScottyFinalizeToken, error) {
	field1Count := 0
	field2Count := 0
	for offset := 0; offset < len(raw); {
		number, wireType, tagLength := protowire.ConsumeTag(raw[offset:])
		if tagLength < 0 {
			return ScottyFinalizeToken{}, fmt.Errorf("invalid Scotty finalize token tag: %w", protowire.ParseError(tagLength))
		}
		value := raw[offset+tagLength:]
		valueLength := protowire.ConsumeFieldValue(number, wireType, value)
		if valueLength < 0 {
			return ScottyFinalizeToken{}, fmt.Errorf("invalid Scotty finalize token field %d: %w", number, protowire.ParseError(valueLength))
		}

		switch number {
		case 1:
			if wireType != protowire.VarintType {
				return ScottyFinalizeToken{}, fmt.Errorf("Scotty finalize token field 1 has wire type %d", wireType)
			}
			version, consumed := protowire.ConsumeVarint(value)
			if consumed != valueLength || version != scottyFinalizeTokenVersion {
				return ScottyFinalizeToken{}, fmt.Errorf("unsupported Scotty finalize token version %d", version)
			}
			field1Count++
		case 2:
			if wireType != protowire.BytesType {
				return ScottyFinalizeToken{}, fmt.Errorf("Scotty finalize token field 2 has wire type %d", wireType)
			}
			opaque, consumed := protowire.ConsumeBytes(value)
			if consumed != valueLength || len(opaque) == 0 {
				return ScottyFinalizeToken{}, fmt.Errorf("Scotty finalize token field 2 is empty or incomplete")
			}
			field2Count++
		}
		offset += tagLength + valueLength
	}

	if field1Count != 1 {
		return ScottyFinalizeToken{}, fmt.Errorf("Scotty finalize token contains %d field-1 values, want 1", field1Count)
	}
	if field2Count != 1 {
		return ScottyFinalizeToken{}, fmt.Errorf("Scotty finalize token contains %d field-2 values, want 1", field2Count)
	}
	return ScottyFinalizeToken{Raw: bytes.Clone(raw)}, nil
}
