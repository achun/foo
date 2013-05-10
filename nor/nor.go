package nor

import (
	"fmt"
)

//调试位标志, 有高位到低位分别表示
//有待确定,阻断执行相关数据,怀疑参数问题,重要步骤,一般信息
var DebugBits = 0

func bug(leve int, i ...interface{}) {
	if DebugBits == 0 || 0 == leve&DebugBits {
		return
	}
	fmt.Println(i...)
}
