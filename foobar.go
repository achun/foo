package foo

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
