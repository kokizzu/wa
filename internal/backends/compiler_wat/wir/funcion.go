// Copyright (C) 2024 武汉凹语言科技有限公司
// SPDX-License-Identifier: AGPL-3.0-or-later

package wir

import "wa-lang.org/wa/internal/backends/compiler_wat/wir/wat"

func (f *Function) ToWatFunc() *wat.Function {
	var wat_func wat.Function

	wat_func.InternalName, wat_func.ExternalName = f.InternalName, f.ExternalName

	for _, r := range f.Results {
		wat_func.Results = append(wat_func.Results, r.Raw()...)
	}

	for _, param := range f.Params {
		raw := param.raw()
		wat_func.Params = append(wat_func.Params, raw...)
	}

	for _, local := range f.Locals {
		raw := local.raw()
		wat_func.Locals = append(wat_func.Locals, raw...)
	}

	wat_func.Insts = f.Insts
	return &wat_func
}
