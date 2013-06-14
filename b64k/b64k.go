package b64k

import (
	"errors"
	"io"
	"net"
)

// B64k
// TCP 长连接交互数据包格式设计
// 2字节数据大小包头+数据
// 2字节包头,chunk size, Big endian 编码
//      0 发送方 Close 完结信号,接收方可以主动Close
//      1..65532    本次chunk数据大小
//      65533       下一个是混入block
//      65534       心跳信号
//      65535       block 结束
type B64k struct {
	b      [16384]byte //接收缓冲区
	c      net.Conn
	size   int //chunk 剩余要读取的大小
	pos    int //保存跨界数据偏移量
	end    int //保存跨界数据结束位置
	closed bool
}

func NewB64k(c net.Conn) *B64k {
	return &B64k{c: c}
}

var (
	EOB       = errors.New("EOB") //end of block
	BOM       = errors.New("MOB") //begin of mixin block
	EOBV      = []byte{0xFF, 0xFF}
	HEARTBEAT = []byte{0xFF, 0xFE}
	MIXIN     = []byte{0xFF, 0xFD}
)

// 解包
// 返回 0 长度[]byte 表示结束或者有 error
func (p *B64k) Read() ([]byte, error) {
	var (
		err     error
		s, h, n int
		ish     bool
	)
	if p.closed {
		return nil, io.EOF
	}
	for {
		if p.pos == 0 {
			s = 0
			n, err = p.c.Read(p.b[0:])
			if err != nil {
				return nil, err
			}
			if n == 0 {
				continue
			}
		} else {
			s = p.pos
			n = p.end
		}
		if p.size == 0 {
			if ish {
				ish = false
				p.size = h + int(p.b[s])
				s++
				h = 0
			} else if n-s == 1 {
				h = int(p.b[s] << 8)
				ish = true
				s++
				continue
			} else {
				p.size = int(p.b[s]<<8) + int(p.b[s+1])
				s += 2
			}
			if n == s {
				continue
			}
		}
		switch p.size {
		case 0:
			p.Close()
			return nil, io.EOF
		case 65535:
			return nil, EOB
		case 65534:
			_, err = p.c.Write(HEARTBEAT)
			if err != nil {
				return nil, err
			}
			continue
		case 65533:
			return nil, BOM
		}
		// 跨边界计算
		p.size -= n - s
		if p.size < 0 {
			p.end = n
			n += p.size
			p.pos = n
			p.size = 0
		} else {
			p.pos = 0
		}
		return p.b[s:n], nil
	}
}
func (p *B64k) Write(b []byte, eob bool) (int, error) {
	var (
		s, e, cnt, size, n int
		err                error
	)
	if p.closed {
		return -1, io.EOF
	}
	max := len(b)
	for {
		if s >= max {
			break
		}
		e = s + 16382
		if e > max {
			size = max - s
			e = max
		}
		if s >= 2 {
			b1, b2 := b[s-2], b[s-1]
			b[s-2] = byte(size >> 8)
			b[s-1] = byte(size)
			n, err = p.write(b[s-2 : e])
			if err == nil {
				b[s-2], b[s-1] = b1, b2
			}
		} else {
			n, err = p.write([]byte{byte(size >> 8), byte(size)})
			if err == nil {
				n, err = p.write(b[s:e])
			}
		}
		if err != nil {
			return cnt, err
		}
		cnt += n
		s += n
	}
	if eob {
		_, err = p.EOB()
	}
	return cnt, err
}

func (p *B64k) write(b []byte) (int, error) {
	if p.closed {
		return -1, io.EOF
	}
	return p.c.Write(b)
}

func (p *B64k) HeartBeat() (int, error) {
	return p.write(HEARTBEAT)
}
func (p *B64k) EOB() (int, error) {
	return p.write(EOBV)
}
func (p *B64k) BOM() (int, error) {
	return p.write(MIXIN)
}

func (p *B64k) Close() {
	if p.closed {
		return
	}
	p.c.Close()
	p.closed = true
}
