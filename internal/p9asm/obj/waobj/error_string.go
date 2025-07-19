// Copyright (C) 2025 武汉凹语言科技有限公司
// SPDX-License-Identifier: AGPL-3.0-or-later

package waobj

type errorString string

func (e errorString) Error() string { return string(e) }
