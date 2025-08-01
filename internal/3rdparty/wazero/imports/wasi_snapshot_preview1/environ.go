package wasi_snapshot_preview1

import (
	"context"

	"wa-lang.org/wa/internal/3rdparty/wazero/api"
	"wa-lang.org/wa/internal/3rdparty/wazero/internal/wasm"
)

const (
	functionEnvironGet      = "environ_get"
	functionEnvironSizesGet = "environ_sizes_get"
)

// environGet is the WASI function named functionEnvironGet that reads
// environment variables.
//
// # Parameters
//
//   - environ: offset to begin writing environment offsets in uint32
//     little-endian encoding to api.Memory
//   - environSizesGet result environc * 4 bytes are written to this offset
//   - environBuf: offset to write the null-terminated variables to api.Memory
//   - the format is like os.Environ: null-terminated "key=val" entries
//   - environSizesGet result environLen bytes are written to this offset
//
// Result (Errno)
//
// The return value is ErrnoSuccess except the following error conditions:
//   - ErrnoFault: there is not enough memory to write results
//
// For example, if environSizesGet wrote environc=2 and environLen=9 for
// environment variables: "a=b", "b=cd" and parameters environ=11 and
// environBuf=1, this function writes the below to api.Memory:
//
//	                              environLen                 uint32le    uint32le
//	             +------------------------------------+     +--------+  +--------+
//	             |                                    |     |        |  |        |
//	  []byte{?, 'a', '=', 'b', 0, 'b', '=', 'c', 'd', 0, ?, 1, 0, 0, 0, 5, 0, 0, 0, ?}
//	environBuf --^                                          ^           ^
//	                             environ offset for "a=b" --+           |
//	                                        environ offset for "b=cd" --+
//
// See environSizesGet
// See https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#environ_get
// See https://en.wikipedia.org/wiki/Null-terminated_string
var environGet = &wasm.HostFunc{
	ExportNames: []string{functionEnvironGet},
	Name:        functionEnvironGet,
	ParamTypes:  []api.ValueType{i32, i32},
	ParamNames:  []string{"environ", "environ_buf"},
	ResultTypes: []api.ValueType{i32},
	Code: &wasm.Code{
		IsHostFunction: true,
		GoFunc:         wasiFunc(environGetFn),
	},
}

func environGetFn(ctx context.Context, mod api.Module, params []uint64) Errno {
	sysCtx := mod.(*wasm.CallContext).Sys
	environ, environBuf := uint32(params[0]), uint32(params[1])

	return writeOffsetsAndNullTerminatedValues(ctx, mod.Memory(), sysCtx.Environ(), environ, environBuf)
}

// environSizesGet is the WASI function named functionEnvironSizesGet that
// reads environment variable sizes.
//
// # Parameters
//
//   - resultEnvironc: offset to write the count of environment variables to
//     api.Memory
//   - resultEnvironvLen: offset to write the null-terminated environment
//     variable length to api.Memory
//
// Result (Errno)
//
// The return value is ErrnoSuccess except the following error conditions:
//   - ErrnoFault: there is not enough memory to write results
//
// For example, if environ are "a=b","b=cd" and parameters resultEnvironc=1 and
// resultEnvironvLen=6, this function writes the below to api.Memory:
//
//	                   uint32le       uint32le
//	                  +--------+     +--------+
//	                  |        |     |        |
//	        []byte{?, 2, 0, 0, 0, ?, 9, 0, 0, 0, ?}
//	 resultEnvironc --^              ^
//		2 variables --+              |
//	             resultEnvironvLen --|
//	    len([]byte{'a','=','b',0,    |
//	           'b','=','c','d',0}) --+
//
// See environGet
// https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#environ_sizes_get
// and https://en.wikipedia.org/wiki/Null-terminated_string
var environSizesGet = &wasm.HostFunc{
	ExportNames: []string{functionEnvironSizesGet},
	Name:        functionEnvironSizesGet,
	ParamTypes:  []api.ValueType{i32, i32},
	ParamNames:  []string{"result.environc", "result.environv_len"},
	ResultTypes: []api.ValueType{i32},
	Code: &wasm.Code{
		IsHostFunction: true,
		GoFunc:         wasiFunc(environSizesGetFn),
	},
}

func environSizesGetFn(ctx context.Context, mod api.Module, params []uint64) Errno {
	sysCtx := mod.(*wasm.CallContext).Sys
	mem := mod.Memory()
	resultEnvironc, resultEnvironvLen := uint32(params[0]), uint32(params[1])

	if !mem.WriteUint32Le(ctx, resultEnvironc, uint32(len(sysCtx.Environ()))) {
		return ErrnoFault
	}
	if !mem.WriteUint32Le(ctx, resultEnvironvLen, sysCtx.EnvironSize()) {
		return ErrnoFault
	}
	return ErrnoSuccess
}
