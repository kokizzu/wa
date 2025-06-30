// Copyright (C) 2024 武汉凹语言科技有限公司
// SPDX-License-Identifier: AGPL-3.0-or-later

// 以不依赖CGO的方式加载不同平台的共享库.
// - linux/macOS: 基于Go的 plugin 标准库
// - windows: 基于 syscall 的 LazyDLL
// - js: 基于 syscal/js, 暂为实现
package dlopen

type LibHandle interface {
	Lookup(symName string) (*Proc, error)
}

func Open(path string) (LibHandle, error) {
	return open(path)
}
