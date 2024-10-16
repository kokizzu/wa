// 版权 @2024 凹语言 作者。保留所有权利。

package mapx

type mapIter struct {
	m   *mapImp
	pos int
}

func MakeMapIter(m *mapImp) *mapIter {
	return &mapIter{m: m}
}

func (this *mapIter) HasNext() (ok bool) {
	return this.pos < len(this.m.values)
}

func (this *mapIter) KeyValue() (k, v interface{}) {
	if this.pos >= len(this.m.values) {
		return nil, nil
	}

	k = this.m.keys[this.pos]
	v = this.m.values[this.pos]
	return
}

func (this *mapIter) Next() (ok bool, k, v interface{}) {
	if this.pos >= len(this.m.values) {
		return
	}

	ok = true
	k = this.m.keys[this.pos]
	v = this.m.values[this.pos]

	this.pos++
	return
}
