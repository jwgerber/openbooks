package server

import "github.com/evan-buss/openbooks/core"

type Repository struct {
	servers core.IrcServers
}

// Reset clears the server list, e.g. when reconnecting or changing channel.
func (r *Repository) Reset() {
	r.servers = core.IrcServers{}
}

func NewRepository() *Repository {
	return &Repository{servers: core.IrcServers{}}
}
