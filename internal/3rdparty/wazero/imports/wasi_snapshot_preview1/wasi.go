// Package wasi_snapshot_preview1 contains Go-defined functions to access
// system calls, such as opening a file, similar to Go's x/sys package. These
// are accessible from WebAssembly-defined functions via importing ModuleName.
// All WASI functions return a single Errno result: ErrnoSuccess on success.
//
// e.g. Call Instantiate before instantiating any wasm binary that imports
// "wasi_snapshot_preview1", Otherwise, it will error due to missing imports.
//
//	ctx := context.Background()
//	r := wazero.NewRuntime(ctx)
//	defer r.Close(ctx) // This closes everything this Runtime created.
//
//	wasi_snapshot_preview1.MustInstantiate(ctx, r)
//	mod, _ := r.InstantiateModuleFromBinary(ctx, wasm)
//
// See https://github.com/WebAssembly/WASI
package wasi_snapshot_preview1

import (
	"context"

	"wa-lang.org/wa/internal/3rdparty/wazero"
	"wa-lang.org/wa/internal/3rdparty/wazero/api"
	"wa-lang.org/wa/internal/3rdparty/wazero/internal/wasm"
)

// ModuleName is the module name WASI functions are exported into.
//
// See https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md
const (
	ModuleName = "wasi_snapshot_preview1"
	i32, i64   = wasm.ValueTypeI32, wasm.ValueTypeI64
)

// MustInstantiate calls Instantiate or panics on error.
//
// This is a simpler function for those who know the module ModuleName is not
// already instantiated, and don't need to unload it.
func MustInstantiate(ctx context.Context, r wazero.Runtime) {
	if _, err := Instantiate(ctx, r); err != nil {
		panic(err)
	}
}

// Instantiate instantiates the ModuleName module into the runtime default
// namespace.
//
// # Notes
//
//   - Failure cases are documented on wazero.Namespace InstantiateModule.
//   - Closing the wazero.Runtime has the same effect as closing the result.
//   - To instantiate into another wazero.Namespace, use NewBuilder instead.
func Instantiate(ctx context.Context, r wazero.Runtime) (api.Closer, error) {
	return NewBuilder(r).Instantiate(ctx, r)
}

// Builder configures the ModuleName module for later use via Compile or Instantiate.
type Builder interface {
	// Compile compiles the ModuleName module that can instantiated in any
	// namespace (wazero.Namespace).
	//
	// Note: This has the same effect as the same function on wazero.HostModuleBuilder.
	Compile(context.Context) (wazero.CompiledModule, error)

	// Instantiate instantiates the ModuleName module into the given namespace.
	//
	// Note: This has the same effect as the same function on wazero.HostModuleBuilder.
	Instantiate(context.Context, wazero.Namespace) (api.Closer, error)
}

// NewBuilder returns a new Builder.
func NewBuilder(r wazero.Runtime) Builder {
	return &builder{r}
}

type builder struct{ r wazero.Runtime }

// hostModuleBuilder returns a new wazero.HostModuleBuilder for ModuleName
func (b *builder) hostModuleBuilder() wazero.HostModuleBuilder {
	ret := b.r.NewHostModuleBuilder(ModuleName)
	exportFunctions(ret)
	return ret
}

// Compile implements Builder.Compile
func (b *builder) Compile(ctx context.Context) (wazero.CompiledModule, error) {
	return b.hostModuleBuilder().Compile(ctx)
}

// Instantiate implements Builder.Instantiate
func (b *builder) Instantiate(ctx context.Context, ns wazero.Namespace) (api.Closer, error) {
	return b.hostModuleBuilder().Instantiate(ctx, ns)
}

// FunctionExporter exports functions into a wazero.HostModuleBuilder.
type FunctionExporter interface {
	ExportFunctions(wazero.HostModuleBuilder)
}

// NewFunctionExporter returns a new FunctionExporter. This is used for the
// following two use cases:
//   - Overriding a builtin function with an alternate implementation.
//   - Exporting functions to the module "wasi_unstable" for legacy code.
//
// # Example of overriding default behavior
//
//	// Export the default WASI functions.
//	wasiBuilder := r.NewHostModuleBuilder(ModuleName)
//	wasi_snapshot_preview1.NewFunctionExporter().ExportFunctions(wasiBuilder)
//
//	// Subsequent calls to NewFunctionBuilder override built-in exports.
//	wasiBuilder.NewFunctionBuilder().
//		WithFunc(func(ctx context.Context, mod api.Module, exitCode uint32) {
//		// your custom logic
//		}).Export("proc_exit")
//
// # Example of using the old module name for WASI
//
//	// Instantiate the current WASI functions under the wasi_unstable
//	// instead of wasi_snapshot_preview1.
//	wasiBuilder := r.NewHostModuleBuilder("wasi_unstable")
//	wasi_snapshot_preview1.NewFunctionExporter().ExportFunctions(wasiBuilder)
//	_, err := wasiBuilder.Instantiate(testCtx, r)
func NewFunctionExporter() FunctionExporter {
	return &functionExporter{}
}

type functionExporter struct{}

// ExportFunctions implements FunctionExporter.ExportFunctions
func (functionExporter) ExportFunctions(builder wazero.HostModuleBuilder) {
	exportFunctions(builder)
}

// ## Translation notes
// ### String
// WebAssembly 1.0 has no string type, so any string input parameter expands to two uint32 parameters: offset
// and length.
//
// ### iovec_array
// `iovec_array` is encoded as two uin32le values (i32): offset and count.
//
// ### Result
// Each result besides Errno is always an uint32 parameter. WebAssembly 1.0 can have up to one result,
// which is already used by Errno. This forces other results to be parameters. A result parameter is a memory
// offset to write the result to. As memory offsets are uint32, each parameter representing a result is uint32.
//
// ### Errno
// The WASI specification is sometimes ambiguous resulting in some runtimes interpreting the same function ways.
// Errno mappings are not defined in WASI, yet, so these mappings are best efforts by maintainers. When in doubt
// about portability, first look at /RATIONALE.md and if needed an issue on
// https://github.com/WebAssembly/WASI/issues
//
// ## Memory
// In WebAssembly 1.0 (20191205), there may be up to one Memory per store, which means api.Memory is always the
// wasm.Store Memories index zero: `store.Memories[0].Buffer`
//
// See https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md
// See https://github.com/WebAssembly/WASI/issues/215
// See https://wwa.w3.org/TR/2019/REC-wasm-core-1-20191205/#memory-instances%E2%91%A0.

// exportFunctions adds all go functions that implement wasi.
// These should be exported in the module named ModuleName.
func exportFunctions(builder wazero.HostModuleBuilder) {
	exporter := builder.(wasm.HostFuncExporter)

	// Note: these are ordered per spec for consistency even if the resulting
	// map can't guarantee that.
	// See https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#functions
	exporter.ExportHostFunc(argsGet)
	exporter.ExportHostFunc(argsSizesGet)
	exporter.ExportHostFunc(environGet)
	exporter.ExportHostFunc(environSizesGet)
	exporter.ExportHostFunc(clockResGet)
	exporter.ExportHostFunc(clockTimeGet)
	exporter.ExportHostFunc(fdAdvise)
	exporter.ExportHostFunc(fdAllocate)
	exporter.ExportHostFunc(fdClose)
	exporter.ExportHostFunc(fdDatasync)
	exporter.ExportHostFunc(fdFdstatGet)
	exporter.ExportHostFunc(fdFdstatSetFlags)
	exporter.ExportHostFunc(fdFdstatSetRights)
	exporter.ExportHostFunc(fdFilestatGet)
	exporter.ExportHostFunc(fdFilestatSetSize)
	exporter.ExportHostFunc(fdFilestatSetTimes)
	exporter.ExportHostFunc(fdPread)
	exporter.ExportHostFunc(fdPrestatGet)
	exporter.ExportHostFunc(fdPrestatDirName)
	exporter.ExportHostFunc(fdPwrite)
	exporter.ExportHostFunc(fdRead)
	exporter.ExportHostFunc(fdReaddir)
	exporter.ExportHostFunc(fdRenumber)
	exporter.ExportHostFunc(fdSeek)
	exporter.ExportHostFunc(fdSync)
	exporter.ExportHostFunc(fdTell)
	exporter.ExportHostFunc(fdWrite)
	exporter.ExportHostFunc(pathCreateDirectory)
	exporter.ExportHostFunc(pathFilestatGet)
	exporter.ExportHostFunc(pathFilestatSetTimes)
	exporter.ExportHostFunc(pathLink)
	exporter.ExportHostFunc(pathOpen)
	exporter.ExportHostFunc(pathReadlink)
	exporter.ExportHostFunc(pathRemoveDirectory)
	exporter.ExportHostFunc(pathRename)
	exporter.ExportHostFunc(pathSymlink)
	exporter.ExportHostFunc(pathUnlinkFile)
	exporter.ExportHostFunc(pollOneoff)
	exporter.ExportHostFunc(procExit)
	exporter.ExportHostFunc(procRaise)
	exporter.ExportHostFunc(schedYield)
	exporter.ExportHostFunc(randomGet)
	exporter.ExportHostFunc(sockAccept)
	exporter.ExportHostFunc(sockRecv)
	exporter.ExportHostFunc(sockSend)
	exporter.ExportHostFunc(sockShutdown)
}

func writeOffsetsAndNullTerminatedValues(ctx context.Context, mem api.Memory, values []string, offsets, bytes uint32) Errno {
	for _, value := range values {
		// Write current offset and advance it.
		if !mem.WriteUint32Le(ctx, offsets, bytes) {
			return ErrnoFault
		}
		offsets += 4 // size of uint32

		// Write the next value to memory with a NUL terminator
		if !mem.Write(ctx, bytes, []byte(value)) {
			return ErrnoFault
		}
		bytes += uint32(len(value))
		if !mem.WriteByte(ctx, bytes, 0) {
			return ErrnoFault
		}
		bytes++
	}

	return ErrnoSuccess
}

// wasiFunc special cases that all WASI functions return a single Errno
// result. The returned value will be written back to the stack at index zero.
type wasiFunc func(ctx context.Context, mod api.Module, params []uint64) Errno

// Call implements the same method as documented on api.GoModuleFunction.
func (f wasiFunc) Call(ctx context.Context, mod api.Module, stack []uint64) {
	// Write the result back onto the stack
	stack[0] = uint64(f(ctx, mod, stack))
}

// stubFunction stubs for GrainLang per #271.
func stubFunction(name string, paramTypes []wasm.ValueType, paramNames []string) *wasm.HostFunc {
	return &wasm.HostFunc{
		Name:        name,
		ExportNames: []string{name},
		ParamTypes:  paramTypes,
		ParamNames:  paramNames,
		ResultTypes: []wasm.ValueType{i32},
		Code: &wasm.Code{
			IsHostFunction: true,
			Body:           []byte{wasm.OpcodeI32Const, byte(ErrnoNosys), wasm.OpcodeEnd},
		},
	}
}
