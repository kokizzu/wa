// Copyright (C) 2025 武汉凹语言科技有限公司
// SPDX-License-Identifier: AGPL-3.0-or-later

package link

import (
	"encoding/binary"
	"io"

	"wa-lang.org/wa/internal/native/abi"
	"wa-lang.org/wa/internal/native/link/elf"
)

// 生成 ELF 格式文件
func LinkELF(prog *abi.LinkedProgram) ([]byte, error) {
	return _LinkELF_RV64(prog)
}

func _LinkELF_RV64(prog *abi.LinkedProgram) ([]byte, error) {
	var (
		ehOff = int64(0)
		phOff = int64(elf.ELF64HDRSIZE)

		textOff = int64(elf.ELF64HDRSIZE + 2*elf.ELF64PHDRSIZE)
		dataOff = textOff + int64(len(prog.TextData))
	)

	var eh elf.ElfHeader64
	copy(eh.Ident[:], elf.ELFMAG)
	eh.Ident[elf.EI_CLASS] = byte(elf.ELFCLASS64)   // 64位
	eh.Ident[elf.EI_DATA] = byte(elf.ELFDATA2LSB)   // 小端序
	eh.Ident[elf.EI_VERSION] = byte(elf.EV_CURRENT) // 文件版本
	eh.Type = uint16(elf.ET_EXEC)                   // 可执行程序
	eh.Machine = uint16(elf.EM_RISCV)               // CPU类型
	eh.Version = uint32(elf.EV_CURRENT)             // ELF版本
	eh.Entry = uint64(prog.TextAddr)                // 程序开始地址
	eh.Phoff = uint64(phOff)                        // 程序头位置
	eh.Shoff = 0                                    // 不写节区表
	eh.Flags = 0                                    // 处理器特殊标志
	eh.Ehsize = elf.ELF64HDRSIZE                    // 文件头大小
	eh.Phentsize = elf.ELF64PHDRSIZE                // 程序头大小
	eh.Phnum = 2                                    // 程序头表中表项的数量(text 和 data)
	eh.Shentsize = 0                                // 节头表中每一个表项的大小
	eh.Shnum = 0                                    // 节头表中表项的数量
	eh.Shstrndx = 0                                 // 节头表中与节名字表相对应的表项的索引

	// 程序头: .text (RX)
	textPh := elf.ElfProgHeader{
		Type:   elf.PT_LOAD,
		Flags:  elf.PF_R | elf.PF_X,        // 可读+执行, 不可修改
		Off:    uint64(textOff),            // 数据段 offset
		Vaddr:  uint64(prog.TextAddr),      // 虚拟内存地址
		Paddr:  uint64(prog.TextAddr),      // 物理内存地址
		Filesz: uint64(len(prog.TextData)), // 数据段文件大小
		Memsz:  uint64(len(prog.TextData)), // 内存大小
		Align:  1,                          // 设为1避免 vaddr/offset 对齐约束
	}

	// 程序头: .data
	dataPh := elf.ElfProgHeader{
		Type:   elf.PT_LOAD,
		Flags:  elf.PF_R | elf.PF_W,        // 可读写
		Off:    uint64(dataOff),            // 数据段 offset
		Vaddr:  uint64(prog.DataAddr),      // 虚拟内存地址
		Paddr:  uint64(prog.DataAddr),      // 物理内存地址
		Filesz: uint64(len(prog.DataData)), // 数据段文件大小
		Memsz:  uint64(len(prog.DataData)), // 内存大小
		Align:  1,                          // 设为1避免 vaddr/offset 对齐约束
	}

	// 构造内存缓存
	var buf _MemBuffer

	// 写 ELF 头
	buf.Seek(ehOff, io.SeekStart)
	if err := binary.Write(&buf, binary.LittleEndian, &eh); err != nil {
		return nil, err
	}

	// 写程序头
	buf.Seek(int64(eh.Phoff), io.SeekStart)
	if err := binary.Write(&buf, binary.LittleEndian, &textPh); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.LittleEndian, &dataPh); err != nil {
		return nil, err
	}

	// 写段内容
	buf.WriteAt(prog.TextData, textOff)
	buf.WriteAt(prog.DataData, dataOff)

	// OK
	return buf.Bytes(), nil
}
