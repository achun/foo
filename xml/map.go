package xml

import (
	"encoding/xml"
	"errors"
)

type XMap struct {
	Name     string `json:"-"`
	Attr     []xml.Attr
	CharData string
	Items    map[string]*XMap
	Parent   *XMap `json:"-"`
}

func MapElement(d *xml.Decoder) *XMap {
	root := XMap{Name: "root"}
	root.Items = map[string]*XMap{}
	var (
		son *XMap
		cd  string
		err error
	)
	deep := 0
	dee := 0
	parent := &root
	for ; err == nil; cd, son, err = next(d, &deep) {
		if son != nil {
			if deep > dee {
				_, ok := parent.Items[son.Name]
				if !ok {
					son.Parent = parent
					son.Items = map[string]*XMap{}
					parent.Items[son.Name] = son
				}
				parent = parent.Items[son.Name]
			}
		} else {
			if deep == dee {
				if len(parent.CharData) == 0 || cd != "" && len(cd) < len(parent.CharData) {
					parent.CharData = cd
				}
			} else if deep < dee {
				parent = parent.Parent
			}
		}
		dee = deep
	}
	return &root
}
func next(d *xml.Decoder, deep *int) (cd string, xt *XMap, err error) {
	t, err := d.RawToken()
	if err != nil {
		return cd, xt, err
	}
	switch t1 := t.(type) {
	case xml.ProcInst:
	case xml.CharData:
		cd = string(t1)
	case xml.StartElement:
		*deep++
		xt = &XMap{Name: t1.Name.Local}
		xt.Attr = t1.Attr
	case xml.EndElement:
		*deep--
	default:
		return cd, xt, errors.New("unkow token")
	}
	return cd, xt, nil
}
