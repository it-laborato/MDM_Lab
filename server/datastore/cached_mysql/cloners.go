package cached_mysql

import (
	"encoding/json"

	"github.com/it-laborato/MDM_Lab/server/mdmlab"
)

type packsList []*mdmlab.Pack

func (pl packsList) Clone() (mdmlab.Cloner, error) {
	var cloned packsList
	if pl == nil {
		return cloned, nil
	}

	cloned = make(packsList, 0, len(pl))
	for _, p := range pl {
		cloned = append(cloned, p.Copy())
	}
	return cloned, nil
}

type rawJSONMessage json.RawMessage

func (r *rawJSONMessage) Clone() (mdmlab.Cloner, error) {
	var clone *rawJSONMessage
	if r == nil {
		return clone, nil
	}

	msg := make(rawJSONMessage, len(*r))
	copy(msg, *r)
	return &msg, nil
}

type integer int

func (i integer) Clone() (mdmlab.Cloner, error) {
	return i, nil
}
