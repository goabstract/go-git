package git

import (
	"gopkg.in/src-d/go-git.v3/clients/common"
	"gopkg.in/src-d/go-git.v4/core"
)

// CloneOptions describe how a clone should be perform
type CloneOptions struct {
	// The (possibly remote) repository URL to clone from
	URL string
	// Auth credentials, if required, to uses with the remote repository
	Auth common.AuthMethod
	// Remote branch to fetch
	ReferenceName core.ReferenceName
}

func (o *CloneOptions) Default() {
	if o.ReferenceName == "" {
		o.ReferenceName = core.HEAD
	}
}

// FetchOptions describe how a fetch should be perform
type FetchOptions struct {
	// Remote branch to fetch
	ReferenceName core.ReferenceName
}

func (o *FetchOptions) Default() {
	if o.ReferenceName == "" {
		o.ReferenceName = core.HEAD
	}
}