package structures

type ListFeedsRequest struct {
	Sort uint8 `codec:"sort"`
}

type ListFeedsResponse struct {
	Count uint64 `codec:"count"`
	Feeds []Feed `codec:"feeds"`
}
