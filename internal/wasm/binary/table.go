package binary

import (
	"bytes"
	"fmt"

	"wa-lang.org/wa/internal/wasm"
)

// decodeTable returns the wasm.Table decoded with the WebAssembly 1.0 (20191205) Binary Format.
//
// See https://www.w3.org/TR/2019/REC-wasm-core-1-20191205/#binary-table
func decodeTable(r *bytes.Reader, enabledFeatures wasm.CoreFeatures) (*wasm.Table, error) {
	tableType, err := r.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("read leading byte: %v", err)
	}

	if tableType != wasm.RefTypeFuncref {
		if err := enabledFeatures.RequireEnabled(wasm.CoreFeatureReferenceTypes); err != nil {
			return nil, fmt.Errorf("table type funcref is invalid: %w", err)
		}
	}

	addrType, min, max, err := decodeLimitsType(r)
	if err != nil {
		return nil, fmt.Errorf("read limits: %v", err)
	}
	if addrType != ValueTypeI32 {
		return nil, fmt.Errorf("table address type must be i32, got %v", addrType)
	}
	if min > wasm.MaximumFunctionIndex {
		return nil, fmt.Errorf("table min must be at most %d", wasm.MaximumFunctionIndex)
	}
	if max != nil {
		if *max < min {
			return nil, fmt.Errorf("table size minimum must not be greater than maximum")
		}
	}
	return &wasm.Table{Min: min, Max: max, Type: tableType}, nil
}

// encodeTable returns the wasm.Table encoded in WebAssembly 1.0 (20191205) Binary Format.
//
// See https://www.w3.org/TR/2019/REC-wasm-core-1-20191205/#binary-table
func encodeTable(i *wasm.Table) []byte {
	return append([]byte{i.Type}, encodeLimitsType(i.Min, i.Max)...)
}
