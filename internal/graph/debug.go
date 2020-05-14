package graph

import "cdr.dev/nfy/internal/clog"

func (ri RecipeIndex) Dump() {
	clog.Debug("begin index dump")
	for i, r := range ri {
		clog.Debug("%v: %v", i, r)
	}
	clog.Debug("end index dump")
}
