package main

import (
	"context"
	"time"

	"git.maharshi.ninja/root/rss2email/structures"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MinimumFrequency == 10min
const MinimumFrequency = 10 * time.Minute

func (c *connection) handleAddFeed(mi *MessageInfo, buf []byte) {
	var req structures.Feed
	ok := c.decodeToInterface(buf, &req)
	if !ok {
		return
	}

	req.CreatedAt = time.Now()
	req.UpdatedAt = time.Now()
	req.ID = primitive.NewObjectID()
	req.Owner = c.userID
	req.Frequency = req.Frequency * time.Second
	req.LastFetched = time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	if req.Frequency < MinimumFrequency {
		req.Frequency = MinimumFrequency
	}

	f, err := c.a.feeds.InsertOne(context.TODO(), req)
	if err != nil {
		c.writeError(mi, structures.ErrorInternal, err)
		return
	}
	c.writeMessage(true, mi, structures.GenericIDResponse{
		OK: true,
		ID: f.InsertedID.(primitive.ObjectID),
	})
}

func (c *connection) handleEditFeed(mi *MessageInfo, buf []byte) {
	var req structures.Feed
	ok := c.decodeToInterface(buf, &req)
	if !ok {
		return
	}
	if req.Frequency < MinimumFrequency {
		req.Frequency = MinimumFrequency
	}
	f, err := c.a.feeds.UpdateOne(context.TODO(), bson.M{
		"_id":      req.ID,
		"owner_id": c.userID,
	}, bson.M{
		"$set": bson.M{
			"name":      req.Name,
			"url":       req.URL,
			"frequency": req.Frequency,
		},
	})
	if err != nil {
		c.writeError(mi, structures.ErrorInternal, err)
		return
	}
	c.writeMessage(true, mi, structures.UpdatedFeedResponse{
		ModifiedCount: uint64(f.ModifiedCount),
	})
}

func (c *connection) handleDeleteFeed(mi *MessageInfo, buf []byte) {
	var req structures.DeleteFeedRequest
	ok := c.decodeToInterface(buf, &req)
	if !ok {
		return
	}
	result, err := c.a.feeds.DeleteOne(context.TODO(), bson.M{
		"_id":      req.ID,
		"owner_id": c.userID,
	})
	if err != nil {
		c.writeError(mi, structures.ErrorInternal, err)
		return
	}
	c.writeMessage(true, mi, structures.DeleteFeedResponse{
		DeletedCount: result.DeletedCount,
	})
}

func (c *connection) handleListFeeds(mi *MessageInfo, buf []byte) {
	var req structures.ListFeedsRequest
	ok := c.decodeToInterface(buf, &req)
	if !ok {
		return
	}

	var feeds []structures.Feed
	cursor, err := c.a.feeds.Find(context.TODO(), bson.M{
		"owner_id": c.userID,
	})
	if err != nil {
		c.writeError(mi, structures.ErrorInternal, err)
		return
	}
	err = cursor.All(context.TODO(), &feeds)
	if err != nil {
		c.writeError(mi, structures.ErrorInternal, err)
		return
	}
	err = cursor.Close(context.TODO())
	if err != nil {
		c.writeError(mi, structures.ErrorInternal, err)
		return
	}

	if feeds == nil {
		feeds = []structures.Feed{}
	}

	c.writeMessage(true, mi, structures.ListFeedsResponse{
		Count: uint64(len(feeds)),
		Feeds: feeds,
	})
}
