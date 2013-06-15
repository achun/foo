package foo

import (
	"fmt"
	"io"
	"log"
)

type Bar interface {
	Bar(Bar) Bar
}

type Foo struct {
	bar Bar
}

func (foo *Foo) Bar(bar Bar) Bar {
	if bar != nil {
		foo.bar = bar
	}
	return foo.bar
}

// 时间格式
const (
	ANSIC       = "Mon Jan _2 15:04:05 2006"
	UnixDate    = "Mon Jan _2 15:04:05 MST 2006"
	Stamp       = "Jan _2 15:04:05"
	StampMilli  = "Jan _2 15:04:05.000"
	StampMicro  = "Jan _2 15:04:05.000000"
	StampNano   = "Jan _2 15:04:05.000000000"
	Kitchen     = "3:04PM"
	RFC822      = "02 Jan 06 15:04 MST"
	RFC822Z     = "02 Jan 06 15:04 -0700"
	RFC3339     = "2006-01-02T15:04:05Z07:00"
	YMD         = "2006-01-02 15:04:05"
	YMDMST      = "2006-01-02 15:04:05 MST"
	YMDZ        = "2006-01-02 15:04:05 -0700"
	RFC3339Nano = "2006-01-02T15:04:05.999999999Z07:00"
	RubyDate    = "Mon Jan 02 15:04:05 -0700 2006"
	RFC1123Z    = "Mon, 02 Jan 2006 15:04:05 -0700"
	RFC850      = "Monday, 02-Jan-06 15:04:05 MST"
	RFC1123     = "Mon, 02 Jan 2006 15:04:05 MST"
)

// 猜测时间字符串风格,
// 返回值:字符串风格，猜测失败返回空字符串
func TimeLayout(s string) string {
	count := make(map[string]int)
	var l int
	var r rune
	for l, r = range s {
		count[string(r)] += 1
	}
	if len(count) == 0 {
		return ""
	}
	if count["_"] == 0 {
		if s[0] >= '0' && s[0] <= '9' {
			switch count[" "] {
			case 0:
				if l < 8 {
					return Kitchen
				}
				if l == 35 {
					return RFC3339Nano
				}
				return RFC3339
			case 1:
				return YMD
			case 2:
				if s[l] > '9' {
					return YMDMST
				}
				return YMDZ
			case 4:
				if s[l] > '9' {
					return RFC822
				}
				return RFC822Z
			}
		} else if count[","] == 0 {
			return RubyDate
		} else if s[l] <= '9' {
			return RFC1123Z
		} else if count["-"] == 0 {
			return RFC1123
		} else {
			return RFC850
		}
	} else {
		switch l {
		case 24:
			return ANSIC
		case 15:
			return Stamp
		case 19:
			return StampMilli
		case 22:
			return StampMicro
		case 25:
			return StampNano
		default:
			return UnixDate
		}
	}
	return ""
}

//在 bs 中查找 delim
//找到返回 delim 的 offset
//找不到返回 len(line)
func Offset(bs []byte, delim byte) int {
	for i, b := range bs {
		if b == delim {
			return i
		}
	}
	return len(bs)
}

func Alert(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
func ErrIfy(v ...interface{}) bool {
	if len(v) == 0 || v[0] == nil || v[0] == io.EOF {
		return false
	}
	log.Println(v...)
	return true
}

func Pln(arg ...interface{}) {
	fmt.Println(arg...)
}
