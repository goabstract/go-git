package progress

import (
	"fmt"
)

const (
	// ScaleGiB is the scale for computing IEC Gibibytes
	ScaleGiB = IECScale(30)
	// ScaleMiB is the scale for computing IEC Mebibytes
	ScaleMiB = IECScale(20)
	// ScaleKiB is the scale for computing IEC Kibibytes
	ScaleKiB = IECScale(10)
	// ScaleBiB is the base scale for representing a number of bytes
	ScaleBiB = IECScale(1)

	// ReceivingObjects represents progress from counting received objects
	ReceivingObjects ProgressType = iota
	// ResolvingDeltas represents progress from counting resolved deltas
	ResolvingDeltas
)

// IECScale is a number used to represent bytes at various scales (GiB, MiB, KiB, bytes)
type IECScale uint64

// ProgressType is the type of progress update being sent
type ProgressType int

// Update is what the consuming code will get notified with
type Update struct {
	Type          ProgressType
	Count         uint32
	Max           uint32
	BytesReceived uint64
	Rate          uint64
}

// Percentage calculates the completion percentage
func (p *Update) Percentage() float32 {
	return 100 * float32(p.Count) / float32(p.Max)
}

// BytesReceivedIEC returns the IEC representation of BytesReceived, e.g. GiB, MiB, KiB
func (p *Update) BytesReceivedIEC() string {
	return ToIECString(p.BytesReceived)
}

// RateIEC returns the IEC representation of Rate, e.g. GiB/s, MiB/s, KiB/s
func (p *Update) RateIEC() string {
	return fmt.Sprintf("%s/s", ToIECString(p.Rate<<10))
}

// String honors the Stringer interface
func (p *Update) String() string {
	t := ""
	if p.Type == ReceivingObjects {
		t = "Receiving objects"
	} else if p.Type == ResolvingDeltas {
		t = "Resolving deltas"
	}
	bytesReceived := ""
	rate := ""
	if p.BytesReceived > 0 {
		if p.Rate > 0 {
			rate = fmt.Sprintf(" | %s", p.RateIEC())
		}
		bytesReceived = fmt.Sprintf(", %s%s", p.BytesReceivedIEC(), rate)
	}
	return fmt.Sprintf("%s: %3.0f%% (%4d/%-4d)%s",
		t,
		p.Percentage(),
		p.Count, p.Max,
		bytesReceived)
}

// ToIEC converts a size of bytes into the corresponding IEC unit
// SEE: https://github.com/git/git/blob/be8661a3286c67a5d4088f4226cbd7f8b76544b0/strbuf.c#L830-L869
func ToIEC(v uint64, scale IECScale) float32 {
	characteristic := uint64(0)
	mantissa := uint64(0)
	result := float32(0.0)

	ib := uint64(1)<<scale - 1

	if scale == ScaleGiB {
		characteristic = v >> scale
		mantissa = ((v & ib) / 10737419)
	} else if scale == ScaleMiB {
		// correction for rounding errors
		val := v + 5243
		characteristic = val >> scale
		mantissa = (((val & ib) * 100) >> scale)
	} else if scale == ScaleKiB {
		// correction for rounding errors
		val := v + 5
		characteristic = val >> scale
		mantissa = (((val & ib) * 100) >> scale)
	} else {
		characteristic = v
	}

	result += float32(characteristic)
	result += float32(mantissa) / float32(100.0)

	return result
}

// ToIECString returns a string representation of a size of bytes including its IEC unit
// Example: 3.14 GiB
func ToIECString(v uint64) string {
	value := float32(0.0)
	unit := ""

	if v > 1<<30 {
		unit = "GiB"
		value = ToIEC(v, ScaleGiB)
	} else if v > 1<<20 {
		unit = "MiB"
		value = ToIEC(v, ScaleMiB)
	} else if v > 1<<10 {
		unit = "KiB"
		value = ToIEC(v, ScaleKiB)
	} else {
		unit = "bytes"
		value = ToIEC(v, ScaleBiB)
	}

	return fmt.Sprintf("%.2f %s", value, unit)
}
