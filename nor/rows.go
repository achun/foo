package nor

import (
	"database/sql"
	"errors"
	"reflect"
	"strconv"
	"time"
)

var EOF = errors.New("End of Rows")

type Rowser interface {
	Err() error
	Close() error
	Closed() bool
	Dst(interface{}) Rowser
	Scan(...interface{}) error
	Rowi(...interface{}) map[string]interface{}
	Row(...interface{}) map[string]reflect.Value
}

type Rows struct {
	err    error
	closed bool
	rows   *sql.Rows
	cols   []string
	dst    reflect.Value
	//列名称 map, 其 int 值从 1 开始,对 Cols 增删或者改变其绝对值会造成不可预计的错误
	//可以通过设置 Cols[key] 为负数(绝对值不能变), 来过滤 Get 返回的 map
	Cols map[string]int
}

//有*sql.Rows和其相关的 struct 对象实例创建 nor.Rows 对象实例
func NewRows(rows *sql.Rows) (ret *Rows, err error) {
	cols, err := rows.Columns()
	if err != nil {
		bug(1<<3, "Rows: NewRows", err)
		rows.Close()
		return
	}
	ret = new(Rows)
	ret.rows = rows
	ret.cols = cols
	ret.Cols = map[string]int{}
	for i, name := range cols {
		ret.Cols[name] = i + 1
	}
	return
}

//返回发生的错误，只遇到任何一次错误，Rows 都会拒绝后续操作并关闭
func (p *Rows) Err() (err error) {
	return p.err
}

func (p *Rows) Closed() bool {
	return p.closed
}

//手动关闭
func (p *Rows) Close() (err error) {
	if p.closed {
		return
	}
	p.closed = true
	return p.rows.Close()
}

//绑定目标 struct 对象实例
//返回值:bool, true 目标对象实例被绑定, false 取消原绑定的目标对象实例
func (p *Rows) Dst(dst interface{}) Rowser {
	rv := reflect.Indirect(reflect.ValueOf(dst))
	ret := rv.IsValid() && rv.Kind() == reflect.Struct
	if ret {
		p.dst = rv
	} else {
		p.dst = reflect.ValueOf(nil)
	}
	return p
}

//支持 struct 对象实例作为参数，如果这样用，该 struct 对象实例比须是唯一的参数
//如果参数违背了参数规则, 将返回错误, 此错误不影响再次 Scan 操作
//如果没有数据设置 error 为 EOF, 并自动关闭
func (p *Rows) Scan(dest ...interface{}) error {
	dst, rv, err := p.scan(dest...)
	if err == nil && rv.IsValid() {
		p.saveStructMapV(dst, rv, false)
	}
	return err
}

//以 map[string]interface{} 形式返回一条记录
//如果遇到错误, 返回 nil, 并自动关闭, 调用 Rows.Err() 查看错误
//如果没有数据, 返回 nil, 并自动关闭, 调用 Rows.Err() 将返回 EOF
func (p *Rows) Rowi(dest ...interface{}) (ret map[string]interface{}) {
	dst, rv, err := p.scan(dest...)
	if err != nil || dst == nil {
		return
	}
	if rv.IsValid() {
		p.saveStructMapV(dst, rv, false)
	}
	ret = make(map[string]interface{})
	for i, name := range p.cols {
		if p.Cols[name] > 0 {
			ret[name] = dst[i]
		}
	}
	return
}

//如果只有一个参数并且是 *struct 类型，以 map[string]reflect.Value 形式返回一条记录
//否则相当于调用 Rows.Scan
//如果遇到错误, 返回 nil, 并自动关闭, 调用 Rows.Err() 查看错误
//如果没有数据, 返回 nil, 并自动关闭, 调用 Rows.Err() 将返回 EOF
func (p *Rows) Row(dest ...interface{}) (ret map[string]reflect.Value) {
	dst, rv, err := p.scan(dest...)
	if err == nil && dst != nil && rv.IsValid() {
		ret = p.saveStructMapV(dst, rv, true)
	}
	return
}

func (p *Rows) canNext() error {
	if p.err != nil {
		return p.err
	}
	if p.closed {
		p.err = errors.New("nor: Rows are closed")
	} else if !p.rows.Next() {
		p.Close()
		p.err = EOF
	}
	return p.err
}

func (p *Rows) errClose() {
	if p.closed {
		return
	}
	if p.err != nil {
		p.Close()
	}
}

func (p *Rows) scan(dest ...interface{}) (dst []interface{}, rv reflect.Value, err error) {
	if len(dest) == 0 && !p.dst.IsValid() {
		p.err = errors.New("nor: Rows expect at least one parameter or Dst")
		return
	}
	err = p.canNext()
	if err != nil {
		return
	}
	cols := p.cols
	if len(dest) == 1 {
		rv = reflect.Indirect(reflect.ValueOf(dest[0]))
	} else if p.dst.IsValid() {
		rv = p.dst
	}

	isStruct := rv.IsValid() && rv.Kind() == reflect.Struct

	if isStruct || len(dest) == 0 {
		dst = make([]interface{}, len(cols))
		for i, _ := range cols {
			var empty interface{}
			dst[i] = &empty
		}
	} else {
		rv = reflect.ValueOf(nil)
		dst = dest
	}
	p.err = p.rows.Scan(dst...)
	p.errClose()
	if p.err != nil {
		bug(1<<3, "Rows.Scan: ", p.err)
		dst = nil
		err = p.err
	}
	return
}
func (p *Rows) saveStructMapV(dst []interface{}, rv reflect.Value, tov bool) (ret map[string]reflect.Value) {
	if tov {
		ret = make(map[string]reflect.Value)
	}
	for i, name := range p.cols {
		title := titleCasedName(name)
		structField := rv.FieldByName(title)
		if !structField.IsValid() {
			bug(1<<2, "Rows.Scan: struct field", title, "invalid")
			continue
		}
		if !structField.CanSet() {
			bug(1<<2, "Rows.Scan: struct field", title, "can not set")
			continue
		}
		scanv := reflect.Indirect(reflect.ValueOf(dst[i])).Elem()
		//类型不同尝试转换
		if scanv.IsValid() && structField.Type() != scanv.Type() {
			scanv = valueToValue(scanv, structField)
		}
		//总是设置mapv
		if tov && p.Cols[name] > 0 {
			ret[name] = scanv
		}
		if !scanv.IsValid() || structField.Type() != scanv.Type() {
			bug(1<<2, "Rows.Scan: struct field", title, "type:", structField.Type(), ",scan Kind:", scanv.Kind())
			continue
		}
		structField.Set(scanv)
	}
	return
}

func valueToValue(srcv reflect.Value, dstv reflect.Value) reflect.Value {
	srcv = reflect.Indirect(srcv)
	st := srcv.Type()
	sk := srcv.Kind()
	si := int(sk)
	dt := dstv.Type()
	dk := dstv.Kind()
	di := int(dk)
	bug(1<<1, "Rows:valueToValue srcv", st, sk, si)
	bug(1<<1, "Rows:valueToValue dstv", dt, dk, di)
	if st == dt {
		return srcv
	}
	var (
		v     reflect.Value
		i64   int64
		u64   uint64
		err   error
		s     string
		asint bool
	)
	//可以转换到字符串
	if dt.String() == "string" && sk == reflect.Slice && st.Elem().Kind() == reflect.Uint8 {
		bs, ok := srcv.Interface().([]byte)
		if ok {
			return reflect.ValueOf(string(bs))
		}
		bug(1<<4, "Rows:", srcv.Interface())
	}
	if si >= int(reflect.Int) && si <= int(reflect.Int64) {
		asint = true
		i64, err = strconv.ParseInt(s, 10, 64)
	}
	if si >= int(reflect.Uint) && si <= int(reflect.Uint64) {
		asint = true
		u64, err = strconv.ParseUint(s, 10, 64)
	}
	if err != nil {
		bug(1<<3, "Rows: valueToValue", err)
		return v
	}
	switch dk {
	case reflect.Ptr:
	case reflect.Invalid:
	case reflect.UnsafePointer:
	case reflect.Func:
	case reflect.Chan:
	case reflect.Interface:
		bug(1<<4, "Rows: valueToValue switch dk", dk)
		return v
	case reflect.Int:
		v = reflect.ValueOf(int(i64))
	case reflect.Int8:
		v = reflect.ValueOf(int8(i64))
	case reflect.Int16:
		v = reflect.ValueOf(int16(i64))
	case reflect.Int32:
		v = reflect.ValueOf(int32(i64))
	case reflect.Int64:
		v = reflect.ValueOf(int64(i64))
	case reflect.Uint:
		v = reflect.ValueOf(uint(u64))
	case reflect.Uint8:
		v = reflect.ValueOf(uint8(u64))
	case reflect.Uint16:
		v = reflect.ValueOf(uint16(u64))
	case reflect.Uint32:
		v = reflect.ValueOf(uint32(u64))
	case reflect.Uint64:
		v = reflect.ValueOf(uint64(u64))
	case reflect.Float32:
		f, err := strconv.ParseFloat(s, 32)
		if err != nil {
			bug(1<<3, "Rows: valueToValue reflect.Float32", err)
		} else {
			v = reflect.ValueOf(f)
		}
	case reflect.Float64:
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			bug(1<<3, "Rows: valueToValue reflect.Float64", err)
		} else {
			v = reflect.ValueOf(f)
		}
	case reflect.Complex64:
	case reflect.Complex128:
		bug(1<<3, "Rows: valueToValue reflect.Complex64/128")
	case reflect.String:
		bug(1<<3, "Rows: valueToValue reflect.String")
	case reflect.Bool:
		if asint {
			v = reflect.ValueOf(i64 != 0 || u64 != uint64(0))
		} else {

		}
	case reflect.Array:
		bug(1<<3, "Rows: valueToValue reflect.Array")
	case reflect.Map:
		bug(1<<3, "Rows: valueToValue reflect.Map")
	case reflect.Slice:
		v = reflect.ValueOf(srcv.Bytes())
	case reflect.Struct:
		//时间类型
		if dt.String() == "time.Time" {
			tim, ok := srcv.Interface().(time.Time)
			if !ok && sk == reflect.Slice && st.String() == "[]uint8" {
				s := string(srcv.Bytes())
				tim, err = time.Parse("2006-01-02 15:04:05", s)
				if err == nil {
					ok = true
				} else {
					bug(1<<4, "Rows: valueToValue reflect.Struct time.Parse", s, err)
				}
			}
			if ok {
				v = reflect.ValueOf(tim)
			}
			if !ok || !v.IsValid() {
				bug(1<<2, "Rows: valueToValue reflect.Struct v.IsValid")
			}
		} else {
			bug(1<<4, "Rows: valueToValue reflect.Struct")
		}
	}
	if !v.IsValid() {
		bug(1<<4, "Rows: valueToValue !v.IsValid() ")
	}
	return v
}

func titleCasedName(name string) string {
	newstr := make([]rune, 0)
	upNextChar := true

	for _, chr := range name {
		switch {
		case upNextChar:
			upNextChar = false
			chr -= ('a' - 'A')
		case chr == '_':
			upNextChar = true
			continue
		}

		newstr = append(newstr, chr)
	}

	return string(newstr)
}
