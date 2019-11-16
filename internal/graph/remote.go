package graph

import (
	"fmt"
	"regexp"
	"strings"
)

type remoteTarget struct {
	// github.com/user/repo
	Repo string
	// wget, curl. Optional
	Target string
	// Tag is master, v1.0.0, etc.
	Tag string
}

var remoteTargetRegex = regexp.MustCompile(`^(?P<repo>[\w-/.]+)(?P<tag>@\S+)?:(?P<target>\S+)$`)

// parseRemoteTarget parses a target like github.com/ammario/dotfiles@master:wget.
func parseRemoteTarget(t string) (*remoteTarget, error) {
	mm := remoteTargetRegex.FindAllStringSubmatch(t, -1)
	if len(mm) != 1 {
		return nil, fmt.Errorf("invalid target (no matches)")
	}
	m := mm[0]
	switch len(m) {
	case 3:
		return &remoteTarget{
			Repo:   m[1],
			Target: m[2],
		}, nil
	case 4:
		return &remoteTarget{
			Repo:   m[1],
			Tag:    strings.TrimPrefix(m[2], "@"),
			Target: m[3],
		}, nil
	default:
		return nil, fmt.Errorf("invalid target (incomplete, parsed %+v)", m)
	}
}
