package structures

import "go.mongodb.org/mongo-driver/bson/primitive"

type ListFeedsRequest struct {
	Sort uint8 `codec:"sort"`
}

type ListFeedsResponse struct {
	Count uint64 `codec:"count"`
	Feeds []Feed `codec:"feeds"`
}

type DeleteFeedRequest struct {
	ID primitive.ObjectID `codec:"id"`
}

type DeleteFeedResponse struct {
	DeletedCount int64 `codec:"deleted_count"`
}

type GenericIDResponse struct {
	OK bool               `codec:"ok"`
	ID primitive.ObjectID `codec:"id"`
}

type UpdatedFeedResponse struct {
	ModifiedCount uint64 `codec:"modified_count"`
}

type VerifyEmailRequest struct {
	Token [32]byte `codec:"token"`
}
