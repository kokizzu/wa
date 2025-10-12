// Copyright (C) 2025 武汉凹语言科技有限公司
// SPDX-License-Identifier: AGPL-3.0-or-later

package token

// 语言类型
type LangType int

const (
	LangType_Unknown LangType = iota
	LangType_Wa               // 凹语言英文
	LangType_Wz               // 凹语言中文
	LangType_Wat              // Wasm 文本
	LangType_Native           // Native 汇编
)

func (lang LangType) String() string {
	switch lang {
	case LangType_Wa:
		return "wa-lang/wa"
	case LangType_Wz:
		return "wa-lang/wz"
	case LangType_Wat:
		return "wa-lang/wat"
	case LangType_Native:
		return "wa-lang/native"
	default:
		return "wa-lang/unknown"
	}
}
