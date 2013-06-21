// blk 读音同 block，是对原始数据进行分块传送/接收的通讯协议
// 例如用于 TCPConn 长连接下的交互通讯。
// 所有方法中返回的 error 有可能是标记状态
//
// Blk 的数据流结构如下
//	block chunk[chunk...]
//	chunk flag[data]
//	flag  uint16
//		0           Close 信号 io.EOF
//		1..65530    chunk data区大小
//		65531       保留 Blk内部使用
//		65532       保留 Blk内部使用
//		65533       后续要插入一个block
//		65534       心跳信号
//		65535       block 结束 EOB
package blk

import (
	"errors"
	"io"
)

type Blk struct {
	r    io.Reader
	w    io.Writer
	size int   //chunk 剩余要读取的大小
	pos  int   //保存跨界数据偏移量
	end  int   //保存跨界数据结束位置
	rerr error //最后一次 r.read 错误
	werr error //最后一次 w.write 错误
	rraw bool  //读出原始流
	wraw bool  //写入原始流
}

func NewBlk(r io.Reader, w io.Writer) *Blk {
	return &Blk{r: r, w: w}
}

var (
	FOB        = errors.New("flag end of block")     // flag
	FOM        = errors.New("flag mixin block")      // flag
	ETE        = errors.New("exceeded the expected") // error:
	EOE        = errors.New("error of empty buffer") // error: zero size of buffer for read/write
	_FOB       = []byte{0xFF, 0xFF}
	_HEARTBEAT = []byte{0xFF, 0xFE}
	_FOM       = []byte{0xFF, 0xFD}
)

// 读取数据到缓冲区，直到填满缓冲区或者遇到标记或错误。
func (p *Blk) Read(b []byte) (int, error) {
	n := 0
	max := len(b)
	for {
		if n >= max {
			return max, nil
		}
		r, err := p.read(b[n:])
		if err != nil {
			return n, err
		}
		if p.rraw {
			return len(r), err
		}
		copy(b[n:], r)
		n += len(r)
	}
	return n, nil
}

// 设置 Read，Write 方法是使用原始数据。
// 原始数据方法可以对两个 Blk 进行 io.Copy。
// 一旦设置使用原始数据，就只能通过 Read，Write 这两个方法进行读取。
// 使用其他方法将产生 ETE 错误
// 使用完毕后要即时重置回去。
func (p *Blk) SetRaw(r bool, w bool) *Blk {
	p.rraw, p.wraw = r, w
	if r {
		p.pos = p.end
	}
	return p
}

var b2 = []byte{'0', '0'}

// 试图读取一个完整 Block
// 此方法向缓冲区填冲数据，直到遇到 EOB 标记或者填满缓冲区。
// 如果缓冲区足够大，FOB将被忽略。反之，会返回 ETE。
// 如果 SetRaw(true,any)，会返回 ETE。
func (p *Blk) ReadBlock(b []byte) (int, error) {
	if p.rraw {
		return 0, ETE
	}
	n, err := p.Read(b)
	if err == FOB {
		return n, nil
	}
	if err != nil {
		return n, err
	}
	_, err = p.read(b2)
	if err == nil {
		return n, ETE
	}
	if err == FOB {
		err = nil
	}
	return n, err
}

// 填满缓冲，或者读完一个 chunk 或者遇到 flag 或者 error
func (p *Blk) read(b []byte) ([]byte, error) {
	var (
		err     error
		s, h, n int
		ish     bool
	)
	if len(b) == 0 {
		return nil, EOE
	}
	for {
		if p.pos == p.end {
			p.pos = 0
			p.end = 0
			s = 0
			n, err = p.readraw(b)
			if err != nil {
				return nil, err
			}
			if p.rraw {
				return b[:n], nil
			}
			if n == 0 {
				continue
			}
			p.end = n
		} else {
			s = p.pos
			n = p.end
		}
		if p.size == 0 {
			if ish {
				ish = false
				p.size = h + int(b[s])
				s++
				h = 0
			} else if n-s == 1 {
				ish = true
				h = int(b[s]) << 8
				p.pos = 0
				p.end = 0
				continue
			} else {
				p.size = int(b[s])<<8 + int(b[s+1])
				s += 2
			}
			p.pos = s
			switch p.size {
			case 0:
				return nil, io.EOF
			case 65535:
				return nil, FOB
			case 65534:
				_, err = p.HeartBeat()
				if err != nil {
					return nil, err
				}
				continue
			case 65533:
				return nil, FOM
			}
		}
		// 只是读取到 p.size
		if n == s {
			continue
		}
		// 已经读入的数据是否有多个 chunk 交叉
		if p.size < n-s {
			p.pos = s + p.size
			n = p.pos
			p.size = 0
		} else {
			// 一个chunk可能还没有读完
			p.size -= n - s
			p.pos = 0
			p.end = 0
		}
		return b[s:n], nil
	}
}

// 写数据
func (p *Blk) Write(b []byte) (int, error) {
	return p.write(b, nil)
}

// 写数据并添加EOB
// 如果 SetRaw(any,true)，会返回 ETE。
func (p *Blk) WriteBlock(b []byte) (int, error) {
	return p.write(b, _FOB)
}

// 从缓冲 b 写数据
func (p *Blk) write(b []byte, raw []byte) (int, error) {
	var (
		s, e, cnt, size, n int
		err                error
	)
	if p.wraw {
		return p.writeraw(b)
	}
	max := len(b)
	for {
		if s >= max {
			break
		}
		e = s + 16382
		if e > max {
			e = max
		}
		size = e - s
		if s == 0 {
			tmp := make([]byte, size+2)
			tmp[0] = byte(size >> 8)
			tmp[1] = byte(size)
			copy(tmp[2:], b[s:e])
			n, err = p.writeraw(tmp)
			if err != nil {
				break
			}
			n -= 2
		} else {
			b1, b2 := b[s-2], b[s-1]
			b[s-2] = byte(size >> 8)
			b[s-1] = byte(size)
			n, err = p.writeraw(b[s-2 : e])
			if err == nil {
				b[s-2], b[s-1] = b1, b2
			}
		}
		if err != nil {
			break
		}
		cnt += n
		s += n
	}
	if err == nil && len(raw) != 0 {
		_, err = p.writeraw(raw)
	}
	return cnt, err
}

func (p *Blk) readraw(b []byte) (int, error) {
	var n int
	if p.rerr != nil {
		return 0, p.rerr
	}
	n, p.rerr = p.r.Read(b)
	return n, p.rerr
}

func (p *Blk) writeraw(b []byte) (int, error) {
	var n int
	if p.werr != nil {
		return 0, p.werr
	}
	n, p.werr = p.w.Write(b)
	return n, p.werr
}

// 写心跳信号
func (p *Blk) HeartBeat() (int, error) {
	if p.wraw {
		return 0, ETE
	}
	return p.writeraw(_HEARTBEAT)
}

// 写 Block 结束标记
func (p *Blk) FOB() (int, error) {
	if p.wraw {
		return 0, ETE
	}
	return p.writeraw(_FOB)
}

// 写混入 Block 标记
func (p *Blk) FOM() (int, error) {
	if p.wraw {
		return 0, ETE
	}
	return p.writeraw(_FOM)
}
