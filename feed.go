package main

import (
	"time"

	"git.maharshi.ninja/root/rss2email/structures"
)

func (c *connection) handleAddFeed(mi *MessageInfo, buf []byte) {

}

func (c *connection) handleListFeeds(mi *MessageInfo, buf []byte) {
	var req structures.ListFeedsRequest
	ok := c.decodeToInterface(buf, &req)
	if !ok {
		return
	}

	c.writeMessage(true, mi, structures.ListFeedsResponse{
		Count: 0xFFFF,
		Feeds: []structures.Feed{
			{
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
				ID:          [12]byte{},
				Owner:       [12]byte{},
				URL:         "AAAAA",
				Frequency:   42069,
				LastFetched: time.Now(),
			},
		},
	})
}
