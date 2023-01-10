package structures

type ErrorCode uint64

const (
	ErrorUnspecified      = 0x0000
	ErrorWhileDecoding    = 0x0010
	ErrorInvalidInputs    = 0x0011
	ErrorInvalidSignature = 0x0012

	ErrorInternal = 0x0101
)
