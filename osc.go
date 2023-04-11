package tcellterm

import (
	"strings"
)

func (vt *VT) osc(data string) {
	selector, val, found := strings.Cut(data, ";")
	if !found {
		return
	}
	switch selector {
	case "0", "2":
		ev := EventTitle{
			EventTerminal: newEventTerminal(vt),
			title:         val,
		}
		vt.postEvent(ev)
	case "8":
		// OSC 8 ; params ; url ST
		// params: key1=value1:key2=value2
		params, url, found := strings.Cut(val, ";")
		if !found {
			return
		}
		for _, param := range strings.Split(params, ":") {
			key, val, found := strings.Cut(param, "=")
			if !found {
				continue
			}
			switch key {
			case "id":
				vt.cursor.attrs = vt.cursor.attrs.UrlId(val)
			}
		}
		vt.cursor.attrs = vt.cursor.attrs.Url(url)
	}
}
