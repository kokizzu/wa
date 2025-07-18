package compiler

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"sync"
	"unsafe"

	"wa-lang.org/wa/internal/3rdparty/wazero/api"
	"wa-lang.org/wa/internal/3rdparty/wazero/experimental"
	"wa-lang.org/wa/internal/3rdparty/wazero/internal/compilationcache"
	"wa-lang.org/wa/internal/3rdparty/wazero/internal/platform"
	"wa-lang.org/wa/internal/3rdparty/wazero/internal/version"
	"wa-lang.org/wa/internal/3rdparty/wazero/internal/wasm"
	"wa-lang.org/wa/internal/3rdparty/wazero/internal/wasmdebug"
	"wa-lang.org/wa/internal/3rdparty/wazero/internal/wasmruntime"
	"wa-lang.org/wa/internal/3rdparty/wazero/internal/wazeroir"
)

type (
	// engine is a Compiler implementation of wasm.Engine
	engine struct {
		enabledFeatures api.CoreFeatures
		codes           map[wasm.ModuleID][]*code // guarded by mutex.
		Cache           compilationcache.Cache
		mux             sync.RWMutex
		// setFinalizer defaults to runtime.SetFinalizer, but overridable for tests.
		setFinalizer  func(obj interface{}, finalizer interface{})
		wazeroVersion string
	}

	// moduleEngine implements wasm.ModuleEngine
	moduleEngine struct {
		// name is the name the module was instantiated with used for error handling.
		name string

		// functions are the functions in a module instances.
		// The index is module instance-scoped. We intentionally avoid using map
		// as the underlying memory region is accessed by assembly directly by using
		// codesElement0Address.
		functions []*function

		importedFunctionCount uint32
	}

	// callEngine holds context per moduleEngine.Call, and shared across all the
	// function calls originating from the same moduleEngine.Call execution.
	callEngine struct {
		// These contexts are read and written by compiled code.
		// Note: structs are embedded to reduce the costs to access fields inside them. Also, this eases field offset
		// calculation.
		moduleContext
		stackContext
		exitContext
		archContext

		// The following fields are not accessed by compiled code directly.

		// stack is the go-allocated stack for holding values and call frames.
		// Note: We never edit len or cap in compiled code, so we won't get screwed when GC comes in.
		//
		// At any point of execution, say currently executing function F2 which was called by F1, then
		// the stack should look like like:
		//
		// 	[..., arg0, arg1, ..., argN, _, _, _, v1, v2, v3, ....
		//	      ^                     {       }
		//	      |                F1's callFrame
		//	      |
		//  stackBasePointer
		//
		// where
		//  - callFrame is the F1's callFrame which called F2. It contains F1's return address, F1's base pointer, and F1's *function.
		//  - stackBasePointer is the stack base pointer stored at (callEngine stackContext.stackBasePointerInBytes)
		//  - arg0, ..., argN are the function parameters, and v1, v2, v3,... are the local variables
		//    including the non-function param locals as well as the temporary variable produced by instructions (e.g i32.const).
		//
		// If the F2 makes a function call to F3 which takes two arguments, then the stack will become:
		//
		// 	[..., arg0, arg1, ..., argN, _, _, _, v1, v2, v3, _, _, _
		//	                            {       }     ^      {       }
		//	                       F1's callFrame     | F2's callFrame
		//	                                          |
		//                                     stackBasePointer
		// where
		// 	- F2's callFrame is pushed above the v2 and v3 (arguments for F3).
		//  - The previous stackBasePointer (pointed at arg0) was saved inside the F2's callFrame.
		//
		// Then, if F3 returns one result, say w1, then the result will look like:
		//
		// 	[..., arg0, arg1, ..., argN, _, _, _, v1, w1, ...
		//	      ^                     {       }
		//	      |                F1's callFrame
		//	      |
		//  stackBasePointer
		//
		// where
		// 	- stackBasePointer was reverted to the position at arg0
		//  - The result from F3 was pushed above v1
		//
		// If the number of parameters is smaller than that of return values, then the empty slots are reserved
		// below the callFrame to store the results on teh return.
		// For example, if F3 takes no parameter but returns N(>0) results, then the stack
		// after making a call against F3 will look like:
		//
		// 	[..., arg0, arg1, ..., argN, _, _, _, v1, v2, v3, res_1, _, res_N, _, _, _
		//	                            {       }            ^                {       }
		//	                       F1's callFrame            |           F2's callFrame
		//	                                                 |
		//                                            stackBasePointer
		// where res_1, ..., res_N are the reserved slots below the call frame. In general,
		// the number of reserved slots equals max(0, len(results)-len(params).
		//
		// This reserved slots are necessary to save the result values onto the stack while not destroying
		// the callFrame value on function returns.
		stack []uint64

		// initialFn is the initial function for this call engine.
		initialFn *function

		// ctx is the context.Context passed to all the host function calls.
		// This is modified when there's a function listener call, otherwise it's always the context.Context
		// passed to the Call API.
		ctx context.Context
		// contextStack is a stack of contexts which is pushed and popped by function listeners.
		// This is used and modified when there are function listeners.
		contextStack *contextStack
	}

	// contextStack is a stack of context.Context.
	contextStack struct {
		self context.Context
		prev *contextStack
	}

	// moduleContext holds the per-function call specific module information.
	// This is subject to be manipulated from compiled native code whenever we make function calls.
	moduleContext struct {
		// fn holds the currently executed *function.
		fn *function

		// moduleInstanceAddress is the address of module instance from which we initialize
		// the following fields. This is set whenever we enter a function or return from function calls.
		//
		// On the entry to the native code, this must be initialized to zero to let native code preamble know
		// that this is the initial function call (which leads to moduleContext initialization pass).
		moduleInstanceAddress uintptr //lint:ignore U1000 This is only used by Compiler code.

		// globalElement0Address is the address of the first element in the global slice,
		// i.e. &ModuleInstance.Globals[0] as uintptr.
		globalElement0Address uintptr
		// memoryElement0Address is the address of the first element in the global slice,
		// i.e. &ModuleInstance.Memory.Buffer[0] as uintptr.
		memoryElement0Address uintptr
		// memorySliceLen is the length of the memory buffer, i.e. len(ModuleInstance.Memory.Buffer).
		memorySliceLen uint64
		// memoryInstance holds the memory instance for this module instance.
		memoryInstance *wasm.MemoryInstance
		// tableElement0Address is the address of the first item in the tables slice,
		// i.e. &ModuleInstance.Tables[0] as uintptr.
		tablesElement0Address uintptr

		// functionsElement0Address is &moduleContext.functions[0] as uintptr.
		functionsElement0Address uintptr

		// typeIDsElement0Address holds the &ModuleInstance.TypeIDs[0] as uintptr.
		typeIDsElement0Address uintptr

		// dataInstancesElement0Address holds the &ModuleInstance.DataInstances[0] as uintptr.
		dataInstancesElement0Address uintptr

		// elementInstancesElement0Address holds the &ModuleInstance.ElementInstances[0] as uintptr.
		elementInstancesElement0Address uintptr
	}

	// stackContext stores the data to access engine.stack.
	stackContext struct {
		// stackPointer on .stack field which is accessed by stack[stackBasePointer+stackBasePointerInBytes*8].
		//
		// Note: stackPointer is not used in assembly since the native code knows exact position of
		// each variable in the value stack from the info from compilation.
		// Therefore, only updated when native code exit from the Compiler world and go back to the Go function.
		stackPointer uint64

		// stackBasePointerInBytes is updated whenever we make function calls.
		// Background: Functions might be compiled as if they use the stack from the bottom.
		// However, in reality, they have to use it from the middle of the stack depending on
		// when these function calls are made. So instead of accessing stack via stackPointer alone,
		// functions are compiled, so they access the stack via [stackBasePointer](fixed for entire function) + [stackPointer].
		// More precisely, stackBasePointer is set to [callee's stack pointer] + [callee's stack base pointer] - [caller's params].
		// This way, compiled functions can be independent of the timing of functions calls made against them.
		stackBasePointerInBytes uint64

		// stackElement0Address is &engine.stack[0] as uintptr.
		// Note: this is updated when growing the stack in builtinFunctionGrowStack.
		stackElement0Address uintptr

		// stackLenInBytes is len(engine.stack[0]) * 8 (bytes).
		// Note: this is updated when growing the stack in builtinFunctionGrowStack.
		stackLenInBytes uint64
	}

	// exitContext will be manipulated whenever compiled native code returns into the Go function.
	exitContext struct {
		// Where we store the status code of Compiler execution.
		statusCode nativeCallStatusCode

		// Set when statusCode == compilerStatusCallBuiltInFunction
		// Indicating the function call index.
		builtinFunctionCallIndex wasm.Index

		// returnAddress is the return address which the engine jumps into
		// after executing a builtin function or host function.
		returnAddress uintptr
	}

	// callFrame holds the information to which the caller function can return.
	// This is mixed in callEngine.stack with other Wasm values just like any other
	// native program (where the stack is the system stack though), and we retrieve the struct
	// with unsafe pointer casts.
	callFrame struct {
		// returnAddress is the return address to which the engine jumps when the callee function returns.
		returnAddress uintptr
		// returnStackBasePointerInBytes is the stack base pointer to set on stackContext.stackBasePointerInBytes
		// when the callee function returns.
		returnStackBasePointerInBytes uint64
		// function is the caller *function, and is used to retrieve the stack trace.
		// Note: should be possible to revive *function from returnAddress, but might be costly.
		function *function
	}

	// Function corresponds to function instance in Wasm, and is created from `code`.
	function struct {
		// codeInitialAddress is the pre-calculated pointer pointing to the initial byte of .codeSegment slice.
		// That mean codeInitialAddress always equals uintptr(unsafe.Pointer(&.codeSegment[0]))
		// and we cache the value (uintptr(unsafe.Pointer(&.codeSegment[0]))) to this field,
		// so we don't need to repeat the calculation on each function call.
		codeInitialAddress uintptr
		// stackPointerCeil is the max of the stack pointer this function can reach. Lazily applied via maybeGrowStack.
		stackPointerCeil uint64
		// source is the source function instance from which this is compiled.
		source *wasm.FunctionInstance
		// moduleInstanceAddress holds the address of source.ModuleInstance.
		moduleInstanceAddress uintptr
		// parent holds code from which this is crated.
		parent *code
	}

	// code corresponds to a function in a module (not instantiated one). This holds the machine code
	// compiled by wazero compiler.
	code struct {
		// codeSegment is holding the compiled native code as a byte slice.
		codeSegment []byte
		// See the doc for codeStaticData type.
		// stackPointerCeil is the max of the stack pointer this function can reach. Lazily applied via maybeGrowStack.
		stackPointerCeil uint64

		// indexInModule is the index of this function in the module. For logging purpose.
		indexInModule wasm.Index
		// sourceModule is the module from which this function is compiled. For logging purpose.
		sourceModule *wasm.Module
		// listener holds a listener to notify when this function is called.
		listener experimental.FunctionListener
	}
)

// createFunction creates a new function which uses the native code compiled.
func (c *code) createFunction(f *wasm.FunctionInstance) *function {
	return &function{
		codeInitialAddress:    uintptr(unsafe.Pointer(&c.codeSegment[0])),
		stackPointerCeil:      c.stackPointerCeil,
		moduleInstanceAddress: uintptr(unsafe.Pointer(f.Module)),
		source:                f,
		parent:                c,
	}
}

// Native code reads/writes Go's structs with the following constants.
// See TestVerifyOffsetValue for how to derive these values.
const (
	// Offsets for moduleEngine.functions
	moduleEngineFunctionsOffset = 16

	// Offsets for callEngine moduleContext.
	callEngineModuleContextFnOffset                              = 0
	callEngineModuleContextModuleInstanceAddressOffset           = 8
	callEngineModuleContextGlobalElement0AddressOffset           = 16
	callEngineModuleContextMemoryElement0AddressOffset           = 24
	callEngineModuleContextMemorySliceLenOffset                  = 32
	callEngineModuleContextMemoryInstanceOffset                  = 40
	callEngineModuleContextTablesElement0AddressOffset           = 48
	callEngineModuleContextFunctionsElement0AddressOffset        = 56
	callEngineModuleContextTypeIDsElement0AddressOffset          = 64
	callEngineModuleContextDataInstancesElement0AddressOffset    = 72
	callEngineModuleContextElementInstancesElement0AddressOffset = 80

	// Offsets for callEngine stackContext.
	callEngineStackContextStackPointerOffset            = 88
	callEngineStackContextStackBasePointerInBytesOffset = 96
	callEngineStackContextStackElement0AddressOffset    = 104
	callEngineStackContextStackLenInBytesOffset         = 112

	// Offsets for callEngine exitContext.
	callEngineExitContextNativeCallStatusCodeOffset     = 120
	callEngineExitContextBuiltinFunctionCallIndexOffset = 124
	callEngineExitContextReturnAddressOffset            = 128

	// Offsets for function.
	functionCodeInitialAddressOffset    = 0
	functionSourceOffset                = 16
	functionModuleInstanceAddressOffset = 24

	// Offsets for wasm.ModuleInstance.
	moduleInstanceGlobalsOffset          = 48
	moduleInstanceMemoryOffset           = 72
	moduleInstanceTablesOffset           = 80
	moduleInstanceEngineOffset           = 136
	moduleInstanceTypeIDsOffset          = 152
	moduleInstanceDataInstancesOffset    = 184
	moduleInstanceElementInstancesOffset = 208

	// Offsets for wasm.TableInstance.
	tableInstanceTableOffset    = 0
	tableInstanceTableLenOffset = 8

	// Offsets for wasm.FunctionInstance.
	functionInstanceTypeIDOffset = 88

	// Offsets for wasm.MemoryInstance.
	memoryInstanceBufferOffset    = 0
	memoryInstanceBufferLenOffset = 8

	// Offsets for wasm.GlobalInstance.
	globalInstanceValueOffset = 8

	// Offsets for Go's interface.
	// https://research.swtch.com/interfaces
	// https://github.com/golang/go/blob/release-branch.go1.17/src/runtime/runtime2.go#L207-L210
	interfaceDataOffset = 8

	// Consts for wasm.DataInstance.
	dataInstanceStructSize = 24

	// Consts for wasm.ElementInstance.
	elementInstanceStructSize = 32

	// pointerSizeLog2 satisfies: 1 << pointerSizeLog2 = sizeOf(uintptr)
	pointerSizeLog2 = 3

	// callFrameDataSizeInUint64 is the size of callFrame struct per 8 bytes (= size of uint64).
	callFrameDataSizeInUint64 = 24 / 8
)

// nativeCallStatusCode represents the result of `nativecall`.
// This is set by the native code.
type nativeCallStatusCode uint32

const (
	// nativeCallStatusCodeReturned means the nativecall reaches the end of function, and returns successfully.
	nativeCallStatusCodeReturned nativeCallStatusCode = iota
	// nativeCallStatusCodeCallGoHostFunction means the nativecall returns to make a host function call.
	nativeCallStatusCodeCallGoHostFunction
	// nativeCallStatusCodeCallBuiltInFunction means the nativecall returns to make a builtin function call.
	nativeCallStatusCodeCallBuiltInFunction
	// nativeCallStatusCodeUnreachable means the function invocation reaches "unreachable" instruction.
	nativeCallStatusCodeUnreachable
	// nativeCallStatusCodeInvalidFloatToIntConversion means an invalid conversion of integer to floats happened.
	nativeCallStatusCodeInvalidFloatToIntConversion
	// nativeCallStatusCodeMemoryOutOfBounds means an out-of-bounds memory access happened.
	nativeCallStatusCodeMemoryOutOfBounds
	// nativeCallStatusCodeInvalidTableAccess means either offset to the table was out of bounds of table, or
	// the target element in the table was uninitialized during call_indirect instruction.
	nativeCallStatusCodeInvalidTableAccess
	// nativeCallStatusCodeTypeMismatchOnIndirectCall means the type check failed during call_indirect.
	nativeCallStatusCodeTypeMismatchOnIndirectCall
	nativeCallStatusIntegerOverflow
	nativeCallStatusIntegerDivisionByZero
)

// causePanic causes a panic with the corresponding error to the nativeCallStatusCode.
func (s nativeCallStatusCode) causePanic() {
	var err error
	switch s {
	case nativeCallStatusIntegerOverflow:
		err = wasmruntime.ErrRuntimeIntegerOverflow
	case nativeCallStatusIntegerDivisionByZero:
		err = wasmruntime.ErrRuntimeIntegerDivideByZero
	case nativeCallStatusCodeInvalidFloatToIntConversion:
		err = wasmruntime.ErrRuntimeInvalidConversionToInteger
	case nativeCallStatusCodeUnreachable:
		err = wasmruntime.ErrRuntimeUnreachable
	case nativeCallStatusCodeMemoryOutOfBounds:
		err = wasmruntime.ErrRuntimeOutOfBoundsMemoryAccess
	case nativeCallStatusCodeInvalidTableAccess:
		err = wasmruntime.ErrRuntimeInvalidTableAccess
	case nativeCallStatusCodeTypeMismatchOnIndirectCall:
		err = wasmruntime.ErrRuntimeIndirectCallTypeMismatch
	}
	panic(err)
}

func (s nativeCallStatusCode) String() (ret string) {
	switch s {
	case nativeCallStatusCodeReturned:
		ret = "returned"
	case nativeCallStatusCodeCallGoHostFunction:
		ret = "call_host_function"
	case nativeCallStatusCodeCallBuiltInFunction:
		ret = "call_builtin_function"
	case nativeCallStatusCodeUnreachable:
		ret = "unreachable"
	case nativeCallStatusCodeInvalidFloatToIntConversion:
		ret = "invalid float to int conversion"
	case nativeCallStatusCodeMemoryOutOfBounds:
		ret = "memory out of bounds"
	case nativeCallStatusCodeInvalidTableAccess:
		ret = "invalid table access"
	case nativeCallStatusCodeTypeMismatchOnIndirectCall:
		ret = "type mismatch on indirect call"
	case nativeCallStatusIntegerOverflow:
		ret = "integer overflow"
	case nativeCallStatusIntegerDivisionByZero:
		ret = "integer division by zero"
	default:
		panic("BUG")
	}
	return
}

// releaseCode is a runtime.SetFinalizer function that munmaps the code.codeSegment.
func releaseCode(compiledFn *code) {
	codeSegment := compiledFn.codeSegment
	if codeSegment == nil {
		return // already released
	}

	// Setting this to nil allows tests to know the correct finalizer function was called.
	compiledFn.codeSegment = nil
	if err := platform.MunmapCodeSegment(codeSegment); err != nil {
		// munmap failure cannot recover, and happen asynchronously on the finalizer thread. While finalizer
		// functions can return errors, they are ignored. To make these visible for troubleshooting, we panic
		// with additional context. module+funcidx should be enough, but if not, we can add more later.
		panic(fmt.Errorf("compiler: failed to munmap code segment for %s.function[%d]: %w", compiledFn.sourceModule.NameSection.ModuleName,
			compiledFn.indexInModule, err))
	}
}

// CompiledModuleCount implements the same method as documented on wasm.Engine.
func (e *engine) CompiledModuleCount() uint32 {
	return uint32(len(e.codes))
}

// DeleteCompiledModule implements the same method as documented on wasm.Engine.
func (e *engine) DeleteCompiledModule(module *wasm.Module) {
	e.deleteCodes(module)
}

// CompileModule implements the same method as documented on wasm.Engine.
func (e *engine) CompileModule(ctx context.Context, module *wasm.Module, listeners []experimental.FunctionListener) error {
	if _, ok, err := e.getCodes(module); ok { // cache hit!
		return nil
	} else if err != nil {
		return err
	}

	irs, err := wazeroir.CompileFunctions(ctx, e.enabledFeatures, callFrameDataSizeInUint64, module)
	if err != nil {
		return err
	}

	importedFuncs := module.ImportFuncCount()
	funcs := make([]*code, len(module.FunctionSection))
	ln := len(listeners)
	for i, ir := range irs {
		var lsn experimental.FunctionListener
		if i < ln {
			lsn = listeners[i]
		}

		funcIndex := wasm.Index(i)
		var compiled *code
		if ir.GoFunc != nil {
			if compiled, err = compileGoDefinedHostFunction(ir, lsn != nil); err != nil {
				def := module.FunctionDefinitionSection[funcIndex+importedFuncs]
				return fmt.Errorf("error compiling host go func[%s]: %w", def.DebugName(), err)
			}
		} else if compiled, err = compileWasmFunction(e.enabledFeatures, ir, lsn != nil); err != nil {
			def := module.FunctionDefinitionSection[funcIndex+importedFuncs]
			return fmt.Errorf("error compiling wasm func[%s]: %w", def.DebugName(), err)
		}

		// As this uses mmap, we need to munmap on the compiled machine code when it's GCed.
		e.setFinalizer(compiled, releaseCode)

		compiled.listener = lsn
		compiled.indexInModule = funcIndex
		compiled.sourceModule = module
		funcs[funcIndex] = compiled
	}
	return e.addCodes(module, funcs)
}

// NewModuleEngine implements the same method as documented on wasm.Engine.
func (e *engine) NewModuleEngine(name string, module *wasm.Module, importedFunctions, moduleFunctions []*wasm.FunctionInstance) (wasm.ModuleEngine, error) {
	imported := len(importedFunctions)
	me := &moduleEngine{
		name:                  name,
		functions:             make([]*function, imported+len(moduleFunctions)),
		importedFunctionCount: uint32(imported),
	}

	for i, f := range importedFunctions {
		cf := f.Module.Engine.(*moduleEngine).functions[f.Idx]
		me.functions[i] = cf
	}

	codes, ok, err := e.getCodes(module)
	if !ok {
		return nil, fmt.Errorf("source module for %s must be compiled before instantiation", name)
	} else if err != nil {
		return nil, err
	}

	for i, c := range codes {
		f := moduleFunctions[i]
		function := c.createFunction(f)
		me.functions[imported+i] = function
	}
	return me, nil
}

// Name implements the same method as documented on wasm.ModuleEngine.
func (e *moduleEngine) Name() string {
	return e.name
}

// FunctionInstanceReference implements the same method as documented on wasm.ModuleEngine.
func (e *moduleEngine) FunctionInstanceReference(funcIndex wasm.Index) wasm.Reference {
	return uintptr(unsafe.Pointer(e.functions[funcIndex]))
}

// CreateFuncElementInstance implements the same method as documented on wasm.ModuleEngine.
func (e *moduleEngine) CreateFuncElementInstance(indexes []*wasm.Index) *wasm.ElementInstance {
	refs := make([]wasm.Reference, len(indexes))
	for i, index := range indexes {
		if index != nil {
			refs[i] = uintptr(unsafe.Pointer(e.functions[*index]))
		}
	}
	return &wasm.ElementInstance{
		References: refs,
		Type:       wasm.RefTypeFuncref,
	}
}

// InitializeFuncrefGlobals implements the same method as documented on wasm.InitializeFuncrefGlobals.
func (e *moduleEngine) InitializeFuncrefGlobals(globals []*wasm.GlobalInstance) {
	for _, g := range globals {
		if g.Type.ValType == wasm.ValueTypeFuncref {
			if int64(g.Val) == wasm.GlobalInstanceNullFuncRefValue {
				g.Val = 0 // Null funcref is expressed as zero.
			} else {
				// Lowers the stored function index into the interpreter specific function's opaque pointer.
				g.Val = uint64(uintptr(unsafe.Pointer(e.functions[g.Val])))
			}
		}
	}
}

func (e *moduleEngine) NewCallEngine(callCtx *wasm.CallContext, f *wasm.FunctionInstance) (ce wasm.CallEngine, err error) {
	// Note: The input parameters are pre-validated, so a compiled function is only absent on close. Updates to
	// code on close aren't locked, neither is this read.
	compiled := e.functions[f.Idx]
	if compiled == nil { // Lazy check the cause as it could be because the module was already closed.
		if err = callCtx.FailIfClosed(); err == nil {
			panic(fmt.Errorf("BUG: %s.func[%d] was nil before close", e.name, f.Idx))
		}
		return
	}

	initStackSize := initialStackSize
	if initialStackSize < compiled.stackPointerCeil {
		initStackSize = compiled.stackPointerCeil * 2
	}
	return e.newCallEngine(initStackSize, compiled), nil
}

// LookupFunction implements the same method as documented on wasm.ModuleEngine.
func (e *moduleEngine) LookupFunction(t *wasm.TableInstance, typeId wasm.FunctionTypeID, tableOffset wasm.Index) (idx wasm.Index, err error) {
	if tableOffset >= uint32(len(t.References)) || t.Type != wasm.RefTypeFuncref {
		err = wasmruntime.ErrRuntimeInvalidTableAccess
		return
	}
	rawPtr := t.References[tableOffset]
	if rawPtr == 0 {
		err = wasmruntime.ErrRuntimeInvalidTableAccess
		return
	}

	tf := functionFromUintptr(rawPtr)
	if tf.source.TypeID != typeId {
		err = wasmruntime.ErrRuntimeIndirectCallTypeMismatch
		return
	}
	idx = tf.source.Idx

	return
}

// functionFromUintptr resurrects the original *function from the given uintptr
// which comes from either funcref table or OpcodeRefFunc instruction.
func functionFromUintptr(ptr uintptr) *function {
	// Wraps ptrs as the double pointer in order to avoid the unsafe access as detected by race detector.
	//
	// For example, if we have (*function)(unsafe.Pointer(ptr)) instead, then the race detector's "checkptr"
	// subroutine wanrs as "checkptr: pointer arithmetic result points to invalid allocation"
	// https://github.com/golang/go/blob/1ce7fcf139417d618c2730010ede2afb41664211/src/runtime/checkptr.go#L69
	var wrapped *uintptr = &ptr
	return *(**function)(unsafe.Pointer(wrapped))
}

// Call implements the same method as documented on wasm.ModuleEngine.
func (ce *callEngine) Call(ctx context.Context, callCtx *wasm.CallContext, params []uint64) (results []uint64, err error) {
	tp := ce.initialFn.source.Type

	paramCount := len(params)
	if tp.ParamNumInUint64 != paramCount {
		return nil, fmt.Errorf("expected %d params, but passed %d", ce.initialFn.source.Type.ParamNumInUint64, paramCount)
	}

	// We ensure that this Call method never panics as
	// this Call method is indirectly invoked by embedders via store.CallFunction,
	// and we have to make sure that all the runtime errors, including the one happening inside
	// host functions, will be captured as errors, not panics.
	defer func() {
		err = ce.deferredOnCall(recover())
		if err == nil {
			// If the module closed during the call, and the call didn't err for another reason, set an ExitError.
			err = callCtx.FailIfClosed()
			// TODO: ^^ Will not fail if the function was imported from a closed module.
		}
	}()

	ce.initializeStack(tp, params)
	ce.execWasmFunction(ctx, callCtx)

	// This returns a safe copy of the results, instead of a slice view. If we
	// returned a re-slice, the caller could accidentally or purposefully
	// corrupt the stack of subsequent calls
	if resultCount := tp.ResultNumInUint64; resultCount > 0 {
		results = make([]uint64, resultCount)
		copy(results, ce.stack[:resultCount])
	}
	return
}

// initializeStack initializes callEngine.stack before entering native code.
//
// The stack must look like, if len(params) < len(results):
//
//	[arg0, arg1, ..., argN, 0, 0, 0, ...
//	                       {       } ^
//	                       callFrame |
//	                                 |
//	                            stackPointer
//
// else:
//
//	[arg0, arg1, ..., argN, _, _, _,  0, 0, 0, ...
//	                      |        | {       }  ^
//	                      |reserved| callFrame  |
//	                      |        |            |
//	                      |-------->       stackPointer
//	                 len(results)-len(params)
//
//		 where we reserve the slots below the callframe with the length len(results)-len(params).
//
// Note: callFrame {  } is zeroed to indicate that the initial "caller" is this callEngine, not the Wasm function.
//
// See callEngine.stack as well.
func (ce *callEngine) initializeStack(tp *wasm.FunctionType, args []uint64) {
	for _, v := range args {
		ce.pushValue(v)
	}

	ce.stackPointer = uint64(callFrameOffset(tp))

	for i := 0; i < callFrameDataSizeInUint64; i++ {
		ce.stack[ce.stackPointer] = 0
		ce.stackPointer++
	}
}

// callFrameOffset returns the offset of the call frame from the stack base pointer.
//
// See the diagram in callEngine.stack.
func callFrameOffset(funcType *wasm.FunctionType) (ret int) {
	ret = funcType.ResultNumInUint64
	if ret < funcType.ParamNumInUint64 {
		ret = funcType.ParamNumInUint64
	}
	return
}

// deferredOnCall takes the recovered value `recovered`, and wraps it
// with the call frame stack traces when not nil. This also resets
// the state of callEngine so that it can be used for the subsequent calls.
//
// This is defined for testability.
func (ce *callEngine) deferredOnCall(recovered interface{}) (err error) {
	if recovered != nil {
		builder := wasmdebug.NewErrorBuilder()

		// Unwinds call frames from the values stack, starting from the
		// current function `ce.fn`, and the current stack base pointer `ce.stackBasePointerInBytes`.
		fn := ce.fn
		stackBasePointer := int(ce.stackBasePointerInBytes >> 3)
		for {
			def := fn.source.Definition
			builder.AddFrame(def.DebugName(), def.ParamTypes(), def.ResultTypes())

			callFrameOffset := callFrameOffset(fn.source.Type)
			if stackBasePointer != 0 {
				frame := *(*callFrame)(unsafe.Pointer(&ce.stack[stackBasePointer+callFrameOffset]))
				fn = frame.function
				stackBasePointer = int(frame.returnStackBasePointerInBytes >> 3)
			} else { // base == 0 means that this was the last call frame stacked.
				break
			}
		}
		err = builder.FromRecovered(recovered)
	}

	// Allows the reuse of CallEngine.
	ce.stackBasePointerInBytes, ce.stackPointer, ce.moduleInstanceAddress = 0, 0, 0
	ce.moduleContext.fn = ce.initialFn
	return
}

func NewEngine(ctx context.Context, enabledFeatures api.CoreFeatures) wasm.Engine {
	return newEngine(ctx, enabledFeatures)
}

func newEngine(ctx context.Context, enabledFeatures api.CoreFeatures) *engine {
	var wazeroVersion string
	if v := ctx.Value(version.WazeroVersionKey{}); v != nil {
		wazeroVersion = v.(string)
	}
	return &engine{
		enabledFeatures: enabledFeatures,
		codes:           map[wasm.ModuleID][]*code{},
		setFinalizer:    runtime.SetFinalizer,
		Cache:           compilationcache.NewFileCache(ctx),
		wazeroVersion:   wazeroVersion,
	}
}

// Do not make this variable as constant, otherwise there would be
// dangerous memory access from native code.
//
// Background: Go has a mechanism called "goroutine stack-shrink" where Go
// runtime shrinks Goroutine's stack when it is GCing. Shrinking means that
// all the contents on the goroutine stack will be relocated by runtime,
// Therefore, the memory address of these contents change undeterministically.
// Not only shrinks, but also Go runtime grows the goroutine stack at any point
// of function call entries, which also might end up relocating contents.
//
// On the other hand, we hold pointers to the data region of value stack and
// call-frame stack slices and use these raw pointers from native code.
// Therefore, it is dangerous if these two stacks are allocated on stack
// as these stack's address might be changed by Goroutine which we cannot
// detect.
//
// By declaring these values as `var`, slices created via `make([]..., var)`
// will never be allocated on stack [1]. This means accessing these slices via
// raw pointers is safe: As of version 1.18, Go's garbage collector never relocates
// heap-allocated objects (aka no compaction of memory [2]).
//
// On Go upgrades, re-validate heap-allocation via `go build -gcflags='-m' ./internal/engine/compiler/...`.
//
//	[1] https://github.com/golang/go/blob/68ecdc2c70544c303aa923139a5f16caf107d955/src/cmd/compile/internal/escape/utils.go#L206-L208
//	[2] https://github.com/golang/go/blob/68ecdc2c70544c303aa923139a5f16caf107d955/src/runtime/mgc.go#L9
//	[3] https://mayurwadekar2.medium.com/escape-analysis-in-golang-ee40a1c064c1
//	[4] https://medium.com/@yulang.chu/go-stack-or-heap-2-slices-which-keep-in-stack-have-limitation-of-size-b3f3adfd6190
var initialStackSize uint64 = 512

func (e *moduleEngine) newCallEngine(stackSize uint64, fn *function) *callEngine {
	ce := &callEngine{
		stack:         make([]uint64, stackSize),
		archContext:   newArchContext(),
		initialFn:     fn,
		moduleContext: moduleContext{fn: fn},
	}

	stackHeader := (*reflect.SliceHeader)(unsafe.Pointer(&ce.stack))
	ce.stackContext = stackContext{
		stackElement0Address: stackHeader.Data,
		stackLenInBytes:      uint64(stackHeader.Len) << 3,
	}
	return ce
}

func (ce *callEngine) popValue() (ret uint64) {
	ce.stackContext.stackPointer--
	ret = ce.stack[ce.stackTopIndex()]
	return
}

func (ce *callEngine) pushValue(v uint64) {
	ce.stack[ce.stackTopIndex()] = v
	ce.stackContext.stackPointer++
}

func (ce *callEngine) stackTopIndex() uint64 {
	return ce.stackContext.stackPointer + (ce.stackContext.stackBasePointerInBytes >> 3)
}

const (
	builtinFunctionIndexMemoryGrow wasm.Index = iota
	builtinFunctionIndexGrowStack
	builtinFunctionIndexTableGrow
	builtinFunctionIndexFunctionListenerBefore
	builtinFunctionIndexFunctionListenerAfter
	// builtinFunctionIndexBreakPoint is internal (only for wazero developers). Disabled by default.
	builtinFunctionIndexBreakPoint
)

func (ce *callEngine) execWasmFunction(ctx context.Context, callCtx *wasm.CallContext) {
	codeAddr := ce.initialFn.codeInitialAddress
	modAddr := ce.initialFn.moduleInstanceAddress
	ce.ctx = ctx

entry:
	{
		// Call into the native code.
		nativecall(codeAddr, uintptr(unsafe.Pointer(ce)), modAddr)

		// Check the status code from Compiler code.
		switch status := ce.exitContext.statusCode; status {
		case nativeCallStatusCodeReturned:
		case nativeCallStatusCodeCallGoHostFunction:
			calleeHostFunction := ce.moduleContext.fn
			base := int(ce.stackBasePointerInBytes >> 3)

			// In the compiler engine, ce.stack has enough capacity for the
			// max of param or result length, so we don't need to grow when
			// there are more results than parameters.
			stackLen := calleeHostFunction.source.Type.ParamNumInUint64
			if resultLen := calleeHostFunction.source.Type.ResultNumInUint64; resultLen > stackLen {
				stackLen = resultLen
			}
			stack := ce.stack[base : base+stackLen]

			fn := calleeHostFunction.source.GoFunc
			switch fn := fn.(type) {
			case api.GoModuleFunction:
				fn.Call(ce.ctx, callCtx.WithMemory(ce.memoryInstance), stack)
			case api.GoFunction:
				fn.Call(ce.ctx, stack)
			}

			codeAddr, modAddr = ce.returnAddress, ce.moduleInstanceAddress
			goto entry
		case nativeCallStatusCodeCallBuiltInFunction:
			caller := ce.moduleContext.fn
			switch ce.exitContext.builtinFunctionCallIndex {
			case builtinFunctionIndexMemoryGrow:
				ce.builtinFunctionMemoryGrow(ce.ctx, caller.source.Module.Memory)
			case builtinFunctionIndexGrowStack:
				ce.builtinFunctionGrowStack(caller.stackPointerCeil)
			case builtinFunctionIndexTableGrow:
				ce.builtinFunctionTableGrow(ce.ctx, caller.source.Module.Tables)
			case builtinFunctionIndexFunctionListenerBefore:
				ce.builtinFunctionFunctionListenerBefore(ce.ctx, caller)
			case builtinFunctionIndexFunctionListenerAfter:
				ce.builtinFunctionFunctionListenerAfter(ce.ctx, caller)
			}
			if false {
				if ce.exitContext.builtinFunctionCallIndex == builtinFunctionIndexBreakPoint {
					runtime.Breakpoint()
				}
			}

			codeAddr, modAddr = ce.returnAddress, ce.moduleInstanceAddress
			goto entry
		default:
			status.causePanic()
		}
	}
}

// callStackCeiling is the maximum WebAssembly call frame stack height. This allows wazero to raise
// wasm.ErrCallStackOverflow instead of overflowing the Go runtime.
//
// The default value should suffice for most use cases. Those wishing to change this can via `go build -ldflags`.
//
// TODO: allows to configure this via context?
var callStackCeiling = uint64(5000000) // in uint64 (8 bytes) == 40000000 bytes in total == 40mb.

func (ce *callEngine) builtinFunctionGrowStack(stackPointerCeil uint64) {
	oldLen := uint64(len(ce.stack))
	if callStackCeiling < oldLen {
		panic(wasmruntime.ErrRuntimeStackOverflow)
	}

	// Extends the stack's length to oldLen*2+stackPointerCeil.
	newLen := oldLen<<1 + (stackPointerCeil)
	newStack := make([]uint64, newLen)
	top := ce.stackTopIndex()
	copy(newStack[:top], ce.stack[:top])
	ce.stack = newStack
	stackHeader := (*reflect.SliceHeader)(unsafe.Pointer(&ce.stack))
	ce.stackContext.stackElement0Address = stackHeader.Data
	ce.stackContext.stackLenInBytes = newLen << 3
}

func (ce *callEngine) builtinFunctionMemoryGrow(ctx context.Context, mem *wasm.MemoryInstance) {
	newPages := ce.popValue()

	if res, ok := mem.Grow(ctx, uint32(newPages)); !ok {
		ce.pushValue(uint64(0xffffffff)) // = -1 in signed 32-bit integer.
	} else {
		ce.pushValue(uint64(res))
	}

	// Update the moduleContext fields as they become stale after the update ^^.
	bufSliceHeader := (*reflect.SliceHeader)(unsafe.Pointer(&mem.Buffer))
	ce.moduleContext.memorySliceLen = uint64(bufSliceHeader.Len)
	ce.moduleContext.memoryElement0Address = bufSliceHeader.Data
}

func (ce *callEngine) builtinFunctionTableGrow(ctx context.Context, tables []*wasm.TableInstance) {
	tableIndex := uint32(ce.popValue())
	table := tables[tableIndex] // verified not to be out of range by the func validation at compilation phase.
	num := ce.popValue()
	ref := ce.popValue()
	res := table.Grow(ctx, uint32(num), uintptr(ref))
	ce.pushValue(uint64(res))
}

func (ce *callEngine) builtinFunctionFunctionListenerBefore(ctx context.Context, fn *function) {
	base := int(ce.stackBasePointerInBytes >> 3)
	listerCtx := fn.parent.listener.Before(ctx, fn.source.Definition, ce.stack[base:base+fn.source.Type.ParamNumInUint64])
	prevStackTop := ce.contextStack
	ce.contextStack = &contextStack{self: ctx, prev: prevStackTop}
	ce.ctx = listerCtx
}

func (ce *callEngine) builtinFunctionFunctionListenerAfter(ctx context.Context, fn *function) {
	base := int(ce.stackBasePointerInBytes >> 3)
	fn.parent.listener.After(ctx, fn.source.Definition, nil, ce.stack[base:base+fn.source.Type.ResultNumInUint64])
	ce.ctx = ce.contextStack.self
	ce.contextStack = ce.contextStack.prev
}

func compileGoDefinedHostFunction(ir *wazeroir.CompilationResult, withListener bool) (*code, error) {
	compiler, err := newCompiler(ir, withListener)
	if err != nil {
		return nil, err
	}

	if err = compiler.compileGoDefinedHostFunction(); err != nil {
		return nil, err
	}

	c, _, err := compiler.compile()
	if err != nil {
		return nil, err
	}

	return &code{codeSegment: c}, nil
}

func compileWasmFunction(_ api.CoreFeatures, ir *wazeroir.CompilationResult, withListener bool) (*code, error) {
	compiler, err := newCompiler(ir, withListener)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize assembly builder: %w", err)
	}

	if err := compiler.compilePreamble(); err != nil {
		return nil, fmt.Errorf("failed to emit preamble: %w", err)
	}

	var skip bool
	for _, op := range ir.Operations {
		// Compiler determines whether skip the entire label.
		// For example, if the label doesn't have any caller,
		// we don't need to generate native code at all as we never reach the region.
		if op.Kind() == wazeroir.OperationKindLabel {
			skip = compiler.compileLabel(op.(*wazeroir.OperationLabel))
		}
		if skip {
			continue
		}

		if false {
			fmt.Printf("compiling op=%s: %s\n", op.Kind(), compiler)
		}
		var err error
		switch o := op.(type) {
		case *wazeroir.OperationLabel:
			// Label op is already handled ^^.
		case *wazeroir.OperationUnreachable:
			err = compiler.compileUnreachable()
		case *wazeroir.OperationBr:
			err = compiler.compileBr(o)
		case *wazeroir.OperationBrIf:
			err = compiler.compileBrIf(o)
		case *wazeroir.OperationBrTable:
			err = compiler.compileBrTable(o)
		case *wazeroir.OperationCall:
			err = compiler.compileCall(o)
		case *wazeroir.OperationCallIndirect:
			err = compiler.compileCallIndirect(o)
		case *wazeroir.OperationDrop:
			err = compiler.compileDrop(o)
		case *wazeroir.OperationSelect:
			err = compiler.compileSelect(o)
		case *wazeroir.OperationPick:
			err = compiler.compilePick(o)
		case *wazeroir.OperationSet:
			err = compiler.compileSet(o)
		case *wazeroir.OperationGlobalGet:
			err = compiler.compileGlobalGet(o)
		case *wazeroir.OperationGlobalSet:
			err = compiler.compileGlobalSet(o)
		case *wazeroir.OperationLoad:
			err = compiler.compileLoad(o)
		case *wazeroir.OperationLoad8:
			err = compiler.compileLoad8(o)
		case *wazeroir.OperationLoad16:
			err = compiler.compileLoad16(o)
		case *wazeroir.OperationLoad32:
			err = compiler.compileLoad32(o)
		case *wazeroir.OperationStore:
			err = compiler.compileStore(o)
		case *wazeroir.OperationStore8:
			err = compiler.compileStore8(o)
		case *wazeroir.OperationStore16:
			err = compiler.compileStore16(o)
		case *wazeroir.OperationStore32:
			err = compiler.compileStore32(o)
		case *wazeroir.OperationMemorySize:
			err = compiler.compileMemorySize()
		case *wazeroir.OperationMemoryGrow:
			err = compiler.compileMemoryGrow()
		case *wazeroir.OperationConstI32:
			err = compiler.compileConstI32(o)
		case *wazeroir.OperationConstI64:
			err = compiler.compileConstI64(o)
		case *wazeroir.OperationConstF32:
			err = compiler.compileConstF32(o)
		case *wazeroir.OperationConstF64:
			err = compiler.compileConstF64(o)
		case *wazeroir.OperationEq:
			err = compiler.compileEq(o)
		case *wazeroir.OperationNe:
			err = compiler.compileNe(o)
		case *wazeroir.OperationEqz:
			err = compiler.compileEqz(o)
		case *wazeroir.OperationLt:
			err = compiler.compileLt(o)
		case *wazeroir.OperationGt:
			err = compiler.compileGt(o)
		case *wazeroir.OperationLe:
			err = compiler.compileLe(o)
		case *wazeroir.OperationGe:
			err = compiler.compileGe(o)
		case *wazeroir.OperationAdd:
			err = compiler.compileAdd(o)
		case *wazeroir.OperationSub:
			err = compiler.compileSub(o)
		case *wazeroir.OperationMul:
			err = compiler.compileMul(o)
		case *wazeroir.OperationClz:
			err = compiler.compileClz(o)
		case *wazeroir.OperationCtz:
			err = compiler.compileCtz(o)
		case *wazeroir.OperationPopcnt:
			err = compiler.compilePopcnt(o)
		case *wazeroir.OperationDiv:
			err = compiler.compileDiv(o)
		case *wazeroir.OperationRem:
			err = compiler.compileRem(o)
		case *wazeroir.OperationAnd:
			err = compiler.compileAnd(o)
		case *wazeroir.OperationOr:
			err = compiler.compileOr(o)
		case *wazeroir.OperationXor:
			err = compiler.compileXor(o)
		case *wazeroir.OperationShl:
			err = compiler.compileShl(o)
		case *wazeroir.OperationShr:
			err = compiler.compileShr(o)
		case *wazeroir.OperationRotl:
			err = compiler.compileRotl(o)
		case *wazeroir.OperationRotr:
			err = compiler.compileRotr(o)
		case *wazeroir.OperationAbs:
			err = compiler.compileAbs(o)
		case *wazeroir.OperationNeg:
			err = compiler.compileNeg(o)
		case *wazeroir.OperationCeil:
			err = compiler.compileCeil(o)
		case *wazeroir.OperationFloor:
			err = compiler.compileFloor(o)
		case *wazeroir.OperationTrunc:
			err = compiler.compileTrunc(o)
		case *wazeroir.OperationNearest:
			err = compiler.compileNearest(o)
		case *wazeroir.OperationSqrt:
			err = compiler.compileSqrt(o)
		case *wazeroir.OperationMin:
			err = compiler.compileMin(o)
		case *wazeroir.OperationMax:
			err = compiler.compileMax(o)
		case *wazeroir.OperationCopysign:
			err = compiler.compileCopysign(o)
		case *wazeroir.OperationI32WrapFromI64:
			err = compiler.compileI32WrapFromI64()
		case *wazeroir.OperationITruncFromF:
			err = compiler.compileITruncFromF(o)
		case *wazeroir.OperationFConvertFromI:
			err = compiler.compileFConvertFromI(o)
		case *wazeroir.OperationF32DemoteFromF64:
			err = compiler.compileF32DemoteFromF64()
		case *wazeroir.OperationF64PromoteFromF32:
			err = compiler.compileF64PromoteFromF32()
		case *wazeroir.OperationI32ReinterpretFromF32:
			err = compiler.compileI32ReinterpretFromF32()
		case *wazeroir.OperationI64ReinterpretFromF64:
			err = compiler.compileI64ReinterpretFromF64()
		case *wazeroir.OperationF32ReinterpretFromI32:
			err = compiler.compileF32ReinterpretFromI32()
		case *wazeroir.OperationF64ReinterpretFromI64:
			err = compiler.compileF64ReinterpretFromI64()
		case *wazeroir.OperationExtend:
			err = compiler.compileExtend(o)
		case *wazeroir.OperationSignExtend32From8:
			err = compiler.compileSignExtend32From8()
		case *wazeroir.OperationSignExtend32From16:
			err = compiler.compileSignExtend32From16()
		case *wazeroir.OperationSignExtend64From8:
			err = compiler.compileSignExtend64From8()
		case *wazeroir.OperationSignExtend64From16:
			err = compiler.compileSignExtend64From16()
		case *wazeroir.OperationSignExtend64From32:
			err = compiler.compileSignExtend64From32()
		case *wazeroir.OperationDataDrop:
			err = compiler.compileDataDrop(o)
		case *wazeroir.OperationMemoryInit:
			err = compiler.compileMemoryInit(o)
		case *wazeroir.OperationMemoryCopy:
			err = compiler.compileMemoryCopy()
		case *wazeroir.OperationMemoryFill:
			err = compiler.compileMemoryFill()
		case *wazeroir.OperationTableInit:
			err = compiler.compileTableInit(o)
		case *wazeroir.OperationTableCopy:
			err = compiler.compileTableCopy(o)
		case *wazeroir.OperationElemDrop:
			err = compiler.compileElemDrop(o)
		case *wazeroir.OperationRefFunc:
			err = compiler.compileRefFunc(o)
		case *wazeroir.OperationTableGet:
			err = compiler.compileTableGet(o)
		case *wazeroir.OperationTableSet:
			err = compiler.compileTableSet(o)
		case *wazeroir.OperationTableGrow:
			err = compiler.compileTableGrow(o)
		case *wazeroir.OperationTableSize:
			err = compiler.compileTableSize(o)
		case *wazeroir.OperationTableFill:
			err = compiler.compileTableFill(o)
		case *wazeroir.OperationV128Const:
			err = compiler.compileV128Const(o)
		case *wazeroir.OperationV128Add:
			err = compiler.compileV128Add(o)
		case *wazeroir.OperationV128Sub:
			err = compiler.compileV128Sub(o)
		case *wazeroir.OperationV128Load:
			err = compiler.compileV128Load(o)
		case *wazeroir.OperationV128LoadLane:
			err = compiler.compileV128LoadLane(o)
		case *wazeroir.OperationV128Store:
			err = compiler.compileV128Store(o)
		case *wazeroir.OperationV128StoreLane:
			err = compiler.compileV128StoreLane(o)
		case *wazeroir.OperationV128ExtractLane:
			err = compiler.compileV128ExtractLane(o)
		case *wazeroir.OperationV128ReplaceLane:
			err = compiler.compileV128ReplaceLane(o)
		case *wazeroir.OperationV128Splat:
			err = compiler.compileV128Splat(o)
		case *wazeroir.OperationV128Shuffle:
			err = compiler.compileV128Shuffle(o)
		case *wazeroir.OperationV128Swizzle:
			err = compiler.compileV128Swizzle(o)
		case *wazeroir.OperationV128AnyTrue:
			err = compiler.compileV128AnyTrue(o)
		case *wazeroir.OperationV128AllTrue:
			err = compiler.compileV128AllTrue(o)
		case *wazeroir.OperationV128BitMask:
			err = compiler.compileV128BitMask(o)
		case *wazeroir.OperationV128And:
			err = compiler.compileV128And(o)
		case *wazeroir.OperationV128Not:
			err = compiler.compileV128Not(o)
		case *wazeroir.OperationV128Or:
			err = compiler.compileV128Or(o)
		case *wazeroir.OperationV128Xor:
			err = compiler.compileV128Xor(o)
		case *wazeroir.OperationV128Bitselect:
			err = compiler.compileV128Bitselect(o)
		case *wazeroir.OperationV128AndNot:
			err = compiler.compileV128AndNot(o)
		case *wazeroir.OperationV128Shr:
			err = compiler.compileV128Shr(o)
		case *wazeroir.OperationV128Shl:
			err = compiler.compileV128Shl(o)
		case *wazeroir.OperationV128Cmp:
			err = compiler.compileV128Cmp(o)
		case *wazeroir.OperationV128AddSat:
			err = compiler.compileV128AddSat(o)
		case *wazeroir.OperationV128SubSat:
			err = compiler.compileV128SubSat(o)
		case *wazeroir.OperationV128Mul:
			err = compiler.compileV128Mul(o)
		case *wazeroir.OperationV128Div:
			err = compiler.compileV128Div(o)
		case *wazeroir.OperationV128Neg:
			err = compiler.compileV128Neg(o)
		case *wazeroir.OperationV128Sqrt:
			err = compiler.compileV128Sqrt(o)
		case *wazeroir.OperationV128Abs:
			err = compiler.compileV128Abs(o)
		case *wazeroir.OperationV128Popcnt:
			err = compiler.compileV128Popcnt(o)
		case *wazeroir.OperationV128Min:
			err = compiler.compileV128Min(o)
		case *wazeroir.OperationV128Max:
			err = compiler.compileV128Max(o)
		case *wazeroir.OperationV128AvgrU:
			err = compiler.compileV128AvgrU(o)
		case *wazeroir.OperationV128Pmin:
			err = compiler.compileV128Pmin(o)
		case *wazeroir.OperationV128Pmax:
			err = compiler.compileV128Pmax(o)
		case *wazeroir.OperationV128Ceil:
			err = compiler.compileV128Ceil(o)
		case *wazeroir.OperationV128Floor:
			err = compiler.compileV128Floor(o)
		case *wazeroir.OperationV128Trunc:
			err = compiler.compileV128Trunc(o)
		case *wazeroir.OperationV128Nearest:
			err = compiler.compileV128Nearest(o)
		case *wazeroir.OperationV128Extend:
			err = compiler.compileV128Extend(o)
		case *wazeroir.OperationV128ExtMul:
			err = compiler.compileV128ExtMul(o)
		case *wazeroir.OperationV128Q15mulrSatS:
			err = compiler.compileV128Q15mulrSatS(o)
		case *wazeroir.OperationV128ExtAddPairwise:
			err = compiler.compileV128ExtAddPairwise(o)
		case *wazeroir.OperationV128FloatPromote:
			err = compiler.compileV128FloatPromote(o)
		case *wazeroir.OperationV128FloatDemote:
			err = compiler.compileV128FloatDemote(o)
		case *wazeroir.OperationV128FConvertFromI:
			err = compiler.compileV128FConvertFromI(o)
		case *wazeroir.OperationV128Dot:
			err = compiler.compileV128Dot(o)
		case *wazeroir.OperationV128Narrow:
			err = compiler.compileV128Narrow(o)
		case *wazeroir.OperationV128ITruncSatFromF:
			err = compiler.compileV128ITruncSatFromF(o)
		default:
			err = errors.New("unsupported")
		}
		if err != nil {
			return nil, fmt.Errorf("operation %s: %w", op.Kind().String(), err)
		}
	}

	c, stackPointerCeil, err := compiler.compile()
	if err != nil {
		return nil, fmt.Errorf("failed to compile: %w", err)
	}

	return &code{codeSegment: c, stackPointerCeil: stackPointerCeil}, nil
}
