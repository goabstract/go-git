package packfile

import (
	"github.com/goabstract/go-git/v5/plumbing"
)

type DeltaBuilder struct {
	dw   *deltaSelector
	objs map[plumbing.Hash]*ObjectToPack
}

func NewDeltaBuilder(dw *deltaSelector, objs map[plumbing.Hash]*ObjectToPack) *DeltaBuilder {
	return &DeltaBuilder{
		dw:   dw,
		objs: objs,
	}
}

func (d *DeltaBuilder) PrepareData() (map[plumbing.Hash]*ObjectToPack, error) {
	for _, obj := range d.objs {
		if err := d.dw.restoreOriginal(obj); err != nil {
			return nil, err
		}

		if err := d.breakChain(obj); err != nil {
			return nil, err
		}
	}

	return d.objs, nil
}

func (d *DeltaBuilder) breakChain(otp *ObjectToPack) error {
	if !otp.Object.Type().IsDelta() {
		return nil
	}

	// Initial ObjectToPack instances might have a delta assigned to Object
	// but no actual base initially. Once Base is assigned to a delta, it means
	// we already fixed it.
	if otp.Base != nil {
		return nil
	}

	do, ok := otp.Object.(plumbing.DeltaObject)
	if !ok {
		// if this is not a DeltaObject, then we cannot retrieve its base,
		// so we have to break the delta chain here.
		return d.dw.undeltify(otp)
	}

	base, ok := d.objs[do.BaseHash()]
	if !ok {
		// The base of the delta is not in our list of objects to pack, so
		// we break the chain.
		return d.dw.undeltify(otp)
	}
	otp.SetDelta(base, otp.Object)
	return nil
}
