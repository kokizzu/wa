package binary

import (
	"bytes"
	"fmt"
	"io"

	"wa-lang.org/wa/internal/3rdparty/wazero/internal/leb128"
	"wa-lang.org/wa/internal/3rdparty/wazero/internal/wasm"
)

const (
	// subsectionIDModuleName contains only the module name.
	subsectionIDModuleName = uint8(0)
	// subsectionIDFunctionNames is a map of indices to function names, in ascending order by function index
	subsectionIDFunctionNames = uint8(1)
	// subsectionIDLocalNames contain a map of function indices to a map of local indices to their names, in ascending
	// order by function and local index
	subsectionIDLocalNames = uint8(2)

	// 扩展特性
	// https://github.com/WebAssembly/wabt/blob/1.0.29/src/binary.h
	// https://github.com/WebAssembly/extended-name-section/blob/main/proposals/extended-name-section/Overview.md

	subsectionIDLabelNames       = uint8(3)
	subsectionIDTypeNames        = uint8(4)
	subsectionIDTableNames       = uint8(5)
	subsectionIDMemoryNames      = uint8(6)
	subsectionIDGlobalNames      = uint8(7)
	subsectionIDElemSegmentNames = uint8(8)
	subsectionIDDataSegmentNames = uint8(9)
	subsectionIDTagNames         = uint8(10)
)

// decodeNameSection deserializes the data associated with the "name" key in SectionIDCustom according to the
// standard:
//
// * ModuleName decode from subsection 0
// * FunctionNames decode from subsection 1
// * LocalNames decode from subsection 2
//
// See https://www.w3.org/TR/2019/REC-wasm-core-1-20191205/#binary-namesec
func decodeNameSection(r *bytes.Reader, limit uint64) (result *wasm.NameSection, err error) {
	// TODO: add leb128 functions that work on []byte and offset. While using a reader allows us to reuse reader-based
	// leb128 functions, it is less efficient, causes untestable code and in some cases more complex vs plain []byte.
	result = &wasm.NameSection{}

	// subsectionID is decoded if known, and skipped if not
	var subsectionID uint8
	// subsectionSize is the length to skip when the subsectionID is unknown
	var subsectionSize uint32
	var bytesRead uint64
	for limit > 0 {
		if subsectionID, err = r.ReadByte(); err != nil {
			if err == io.EOF {
				return result, nil
			}
			// TODO: untestable as this can't fail for a reason beside EOF reading a byte from a buffer
			return nil, fmt.Errorf("failed to read a subsection ID: %w", err)
		}
		limit--

		if subsectionSize, bytesRead, err = leb128.DecodeUint32(r); err != nil {
			return nil, fmt.Errorf("failed to read the size of subsection[%d]: %w", subsectionID, err)
		}
		limit -= bytesRead

		switch subsectionID {
		case subsectionIDModuleName:
			if result.ModuleName, _, err = decodeUTF8(r, "module name"); err != nil {
				return nil, err
			}
		case subsectionIDFunctionNames:
			if result.FunctionNames, err = decodeFunctionNames(r); err != nil {
				return nil, err
			}
		case subsectionIDLocalNames:
			if result.LocalNames, err = decodeLocalNames(r); err != nil {
				return nil, err
			}
		default: // Skip other subsections.

			// 扩展特性
			// https://github.com/WebAssembly/wabt/blob/1.0.29/src/binary.h
			// https://github.com/WebAssembly/extended-name-section/blob/main/proposals/extended-name-section/Overview.md

			switch subsectionID {
			case subsectionIDLabelNames:
				if _, err = io.CopyN(io.Discard, r, int64(subsectionSize)); err != nil {
					return nil, fmt.Errorf("failed to skip subsection[%d]: %w", subsectionID, err)
				}
			case subsectionIDTypeNames:
				if result.TypeNames, err = decodeExtendedNames(r, subsectionID); err != nil {
					return nil, err
				}
			case subsectionIDTableNames:
				if result.TableNames, err = decodeExtendedNames(r, subsectionID); err != nil {
					return nil, err
				}
			case subsectionIDMemoryNames:
				if result.MemoryNames, err = decodeExtendedNames(r, subsectionID); err != nil {
					return nil, err
				}
			case subsectionIDGlobalNames:
				if result.GlobalNames, err = decodeExtendedNames(r, subsectionID); err != nil {
					return nil, err
				}
			case subsectionIDElemSegmentNames:
				if result.ElemSegmentNames, err = decodeExtendedNames(r, subsectionID); err != nil {
					return nil, err
				}
			case subsectionIDDataSegmentNames:
				if result.DataSegmentNames, err = decodeExtendedNames(r, subsectionID); err != nil {
					return nil, err
				}
			case subsectionIDTagNames:
				if result.TagNames, err = decodeExtendedNames(r, subsectionID); err != nil {
					return nil, err
				}
			default:
				// Note: Not Seek because it doesn't err when given an offset past EOF. Rather, it leads to undefined state.
				if _, err = io.CopyN(io.Discard, r, int64(subsectionSize)); err != nil {
					return nil, fmt.Errorf("failed to skip subsection[%d]: %w", subsectionID, err)
				}
			}
		}
		limit -= uint64(subsectionSize)
	}
	return
}

func decodeFunctionNames(r *bytes.Reader) (wasm.NameMap, error) {
	functionCount, err := decodeFunctionCount(r, subsectionIDFunctionNames)
	if err != nil {
		return nil, err
	}

	result := make(wasm.NameMap, functionCount)
	for i := uint32(0); i < functionCount; i++ {
		functionIndex, err := decodeFunctionIndex(r, subsectionIDFunctionNames)
		if err != nil {
			return nil, err
		}

		name, _, err := decodeUTF8(r, "function[%d] name", functionIndex)
		if err != nil {
			return nil, err
		}
		result[i] = &wasm.NameAssoc{Index: functionIndex, Name: name}
	}
	return result, nil
}

func decodeLocalNames(r *bytes.Reader) (wasm.IndirectNameMap, error) {
	functionCount, err := decodeFunctionCount(r, subsectionIDLocalNames)
	if err != nil {
		return nil, err
	}

	result := make(wasm.IndirectNameMap, functionCount)
	for i := uint32(0); i < functionCount; i++ {
		functionIndex, err := decodeFunctionIndex(r, subsectionIDLocalNames)
		if err != nil {
			return nil, err
		}

		localCount, _, err := leb128.DecodeUint32(r)
		if err != nil {
			return nil, fmt.Errorf("failed to read the local count for function[%d]: %w", functionIndex, err)
		}

		locals := make(wasm.NameMap, localCount)
		for j := uint32(0); j < localCount; j++ {
			localIndex, _, err := leb128.DecodeUint32(r)
			if err != nil {
				return nil, fmt.Errorf("failed to read a local index of function[%d]: %w", functionIndex, err)
			}

			name, _, err := decodeUTF8(r, "function[%d] local[%d] name", functionIndex, localIndex)
			if err != nil {
				return nil, err
			}
			locals[j] = &wasm.NameAssoc{Index: localIndex, Name: name}
		}
		result[i] = &wasm.NameMapAssoc{Index: functionIndex, NameMap: locals}
	}
	return result, nil
}

func decodeExtendedNames(r *bytes.Reader, subsectionIDExtendedNames uint8) (wasm.NameMap, error) {
	namesCount, err := decodeEntendNamesCount(r, subsectionIDExtendedNames)
	if err != nil {
		return nil, err
	}

	result := make(wasm.NameMap, namesCount)
	for i := uint32(0); i < namesCount; i++ {
		idx, err := decodeEntendNameIndex(r, subsectionIDExtendedNames)
		if err != nil {
			return nil, err
		}

		name, _, err := decodeUTF8(r, "subsectionExtendedNames[%d] [%d] name",
			subsectionIDExtendedNames, idx,
		)
		if err != nil {
			return nil, err
		}
		result[i] = &wasm.NameAssoc{Index: idx, Name: name}
	}
	return result, nil
}

func decodeFunctionIndex(r *bytes.Reader, subsectionID uint8) (uint32, error) {
	functionIndex, _, err := leb128.DecodeUint32(r)
	if err != nil {
		return 0, fmt.Errorf("failed to read a function index in subsection[%d]: %w", subsectionID, err)
	}
	return functionIndex, nil
}

func decodeFunctionCount(r *bytes.Reader, subsectionID uint8) (uint32, error) {
	functionCount, _, err := leb128.DecodeUint32(r)
	if err != nil {
		return 0, fmt.Errorf("failed to read the function count of subsection[%d]: %w", subsectionID, err)
	}
	return functionCount, nil
}

func decodeEntendNamesCount(r *bytes.Reader, subsectionID uint8) (uint32, error) {
	n, _, err := leb128.DecodeUint32(r)
	if err != nil {
		return 0, fmt.Errorf("failed to read the extend names count of subsection[%d]: %w", subsectionID, err)
	}
	return n, nil
}

func decodeEntendNameIndex(r *bytes.Reader, subsectionID uint8) (uint32, error) {
	idx, _, err := leb128.DecodeUint32(r)
	if err != nil {
		return 0, fmt.Errorf("failed to read a extend name index in subsection[%d]: %w", subsectionID, err)
	}
	return idx, nil
}

// encodeNameSectionData serializes the data for the "name" key in wasm.SectionIDCustom according to the
// standard:
//
// Note: The result can be nil because this does not encode empty subsections
//
// See https://www.w3.org/TR/2019/REC-wasm-core-1-20191205/#binary-namesec
func encodeNameSectionData(n *wasm.NameSection) (data []byte) {
	if n.ModuleName != "" {
		data = append(data, encodeNameSubsection(subsectionIDModuleName, encodeSizePrefixed([]byte(n.ModuleName)))...)
	}
	if fd := encodeFunctionNameData(n); len(fd) > 0 {
		data = append(data, encodeNameSubsection(subsectionIDFunctionNames, fd)...)
	}
	if ld := encodeLocalNameData(n); len(ld) >= 0 {
		data = append(data, encodeNameSubsection(subsectionIDLocalNames, ld)...)
	}

	// 扩展特性
	// https://github.com/WebAssembly/wabt/blob/1.0.29/src/binary.h
	// https://github.com/WebAssembly/extended-name-section/blob/main/proposals/extended-name-section/Overview.md

	if data := encodeEntendNameData(n.LabelNames); len(data) > 0 {
		// todo: label
	}
	if data := encodeEntendNameData(n.TypeNames); len(data) > 0 {
		data = append(data, encodeNameSubsection(subsectionIDTypeNames, data)...)
	}
	if data := encodeEntendNameData(n.TableNames); len(data) > 0 {
		data = append(data, encodeNameSubsection(subsectionIDTableNames, data)...)
	}
	if data := encodeEntendNameData(n.MemoryNames); len(data) > 0 {
		data = append(data, encodeNameSubsection(subsectionIDMemoryNames, data)...)
	}
	if data := encodeEntendNameData(n.GlobalNames); len(data) > 0 {
		data = append(data, encodeNameSubsection(subsectionIDGlobalNames, data)...)
	}
	if data := encodeEntendNameData(n.ElemSegmentNames); len(data) > 0 {
		data = append(data, encodeNameSubsection(subsectionIDElemSegmentNames, data)...)
	}
	if data := encodeEntendNameData(n.DataSegmentNames); len(data) > 0 {
		data = append(data, encodeNameSubsection(subsectionIDDataSegmentNames, data)...)
	}
	if data := encodeEntendNameData(n.TagNames); len(data) > 0 {
		data = append(data, encodeNameSubsection(subsectionIDTagNames, data)...)
	}

	return
}

// encodeFunctionNameData encodes the data for the function name subsection.
// See https://www.w3.org/TR/2019/REC-wasm-core-1-20191205/#binary-funcnamesec
func encodeFunctionNameData(n *wasm.NameSection) []byte {
	if len(n.FunctionNames) == 0 {
		return nil
	}

	return encodeNameMap(n.FunctionNames)
}

func encodeEntendNameData(nameMap wasm.NameMap) []byte {
	if len(nameMap) == 0 {
		return nil
	}

	return encodeNameMap(nameMap)
}

func encodeNameMap(m wasm.NameMap) []byte {
	count := uint32(len(m))
	data := leb128.EncodeUint32(count)
	for _, na := range m {
		data = append(data, encodeNameAssoc(na)...)
	}
	return data
}

// encodeLocalNameData encodes the data for the local name subsection.
// See https://www.w3.org/TR/2019/REC-wasm-core-1-20191205/#binary-localnamesec
func encodeLocalNameData(n *wasm.NameSection) []byte {
	if len(n.LocalNames) == 0 {
		// 保持和 wabt-1.0.29/wat2wasm 行为一致
	}

	funcNameCount := uint32(len(n.LocalNames))
	subsection := leb128.EncodeUint32(funcNameCount)

	for _, na := range n.LocalNames {
		locals := encodeNameMap(na.NameMap)
		subsection = append(subsection, append(leb128.EncodeUint32(na.Index), locals...)...)
	}
	return subsection
}

// encodeNameSubsection returns a buffer encoding the given subsection
// See https://www.w3.org/TR/2019/REC-wasm-core-1-20191205/#subsections%E2%91%A0
func encodeNameSubsection(subsectionID uint8, content []byte) []byte {
	contentSizeInBytes := leb128.EncodeUint32(uint32(len(content)))
	result := []byte{subsectionID}
	result = append(result, contentSizeInBytes...)
	result = append(result, content...)
	return result
}

// encodeNameAssoc encodes the index and data prefixed by their size.
// See https://www.w3.org/TR/2019/REC-wasm-core-1-20191205/#binary-namemap
func encodeNameAssoc(na *wasm.NameAssoc) []byte {
	return append(leb128.EncodeUint32(na.Index), encodeSizePrefixed([]byte(na.Name))...)
}

// encodeSizePrefixed encodes the data prefixed by their size.
func encodeSizePrefixed(data []byte) []byte {
	size := leb128.EncodeUint32(uint32(len(data)))
	return append(size, data...)
}
