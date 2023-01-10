//go:generate codecgen -o structures.generated.go -j=false -d=42 structures.go
package structures

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	CreatedAt time.Time          `codec:"created_at" bson:"created_at"`
	UpdatedAt time.Time          `codec:"updated_at" bson:"updated_at"`
	ID        primitive.ObjectID `codec:"id" bson:"_id"`
	Address   [20]byte           `codec:"addr" bson:"addr"`
	Email     string             `codec:"email" bson:"email"`
}

type Feed struct {
	CreatedAt   time.Time          `codec:"created_at" bson:"created_at"`
	UpdatedAt   time.Time          `codec:"updated_at" bson:"updated_at"`
	ID          primitive.ObjectID `codec:"id" bson:"_id"`
	Owner       primitive.ObjectID `codec:"owner_id" bson:"owner_id"`
	URL         string             `codec:"feed_url" bson:"feed_url"`
	Frequency   time.Duration      `codec:"frequency" bson:"frequency"`
	LastFetched time.Time          `codec:"last_fetched" bson:"last_fetched"`
}

type SeenItem struct {
	ID        primitive.ObjectID `codec:"id" bson:"_id"`
	FeedID    primitive.ObjectID `codec:"feed_id" bson:"feed_id"`
	GUID      string             `codec:"guid" bson:"guid"`
	Timestamp time.Time          `codec:"timestamp" bson:"timestamp"`
}
