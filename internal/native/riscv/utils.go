// Copyright (C) 2025 武汉凹语言科技有限公司
// SPDX-License-Identifier: AGPL-3.0-or-later

package riscv

import "fmt"

func newU32(v uint32) *uint32 {
	return &v
}

// 输入一个 32 位有符号立即数 imm, 输出 low(12bit)/high(20bit)
// 满足 imm 约等于 (high << 12) + low, 以便于进行长地址跳转的拆分
func split32BitImmediate(imm int64) (low12bit, high20bit int64, err error) {
	// 确保 imm 在 32 位有符号整数范围内
	if err := immIFitsIntN(imm, 32); err != nil {
		return 0, 0, err
	}

	// 如果 imm 能直接放进 12 位 signed 范围([-2048, 2047])
	// 则没必要分拆, low=imm, high=0
	if err := immIFitsIntN(imm, 12); err == nil {
		return imm, 0, nil
	}

	// 先粗略地取高 20 位
	high20bit = imm >> 12

	// 低 12 位是有符号数, 可能是负的
	// 当 imm 的 bit[11]=1 时, 说明低 12 位是负数, 这时 high++ 来补偿
	if imm&(1<<11) != 0 {
		high20bit++
	}

	// 把 low 作为 12 位有符号数扩展
	low12bit = i64SignExtend(imm, 12)

	// 把 high 作为 20 位有符号数扩展
	high20bit = i64SignExtend(high20bit, 20)

	return low12bit, high20bit, nil
}

// 检查 x 是否能装进 nbits 位的有符号整数
func immIFitsIntN(x int64, nbits uint) error {
	nbits--
	min := int64(-1) << nbits
	max := int64(1)<<nbits - 1
	if x < min || x > max {
		if nbits <= 16 {
			return fmt.Errorf("signed immediate %d must be in range [%d, %d] (%d bits)", x, min, max, nbits)
		}
		return fmt.Errorf("signed immediate %#x must be in range [%#x, %#x] (%d bits)", x, min, max, nbits)
	}
	return nil
}

// 把 val 的低 bit 位当作一个有符号数扩展成 int64
func i64SignExtend(val int64, bit uint) int64 {
	// 1. 先左移, 把符号位移到最高位
	// 2. 再算术右移(保持符号), 补全剩余的高位
	return val << (64 - bit) >> (64 - bit)
}
