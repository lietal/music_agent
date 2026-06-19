package tme

import (
	"context"
	"fmt"
	"time"
)

func (c *Client) GetHotComments(ctx context.Context, songID string, limit int) ([]Comment, error) {
	mid := extractMID(songID)

	resp, err := c.Call(ctx, map[string]MusicuSubRequest{
		"req_0": {
			Module: "music.comment.CommentRead",
			Method: "get_hot_comment",
			Param: map[string]any{
				"song_id": mid,
				"num":     limit,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	sub, ok := resp.Req["req_0"]
	if !ok || sub.Code != 0 {
		return nil, fmt.Errorf("hot comments failed: code=%d", sub.Code)
	}

	return parseComments(sub.Data, songID, limit), nil
}

func (c *Client) GetTrackComments(ctx context.Context, songID, commentType string, limit, page int, cursor string) ([]Comment, error) {
	mid := extractMID(songID)

	method := "get_hot_comment"
	switch commentType {
	case "new":
		method = "get_new_comment"
	case "recommend":
		method = "get_recommend_comment"
	case "moment":
		method = "get_moment_comment"
	}

	resp, err := c.Call(ctx, map[string]MusicuSubRequest{
		"req_0": {
			Module: "music.comment.CommentRead",
			Method: method,
			Param: map[string]any{
				"song_id":     mid,
				"num":         limit,
				"page":        page,
				"last_comment_id": cursor,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	sub, ok := resp.Req["req_0"]
	if !ok || sub.Code != 0 {
		return nil, fmt.Errorf("track comments failed: code=%d", sub.Code)
	}

	return parseComments(sub.Data, songID, limit), nil
}

func parseComments(data map[string]any, songID string, limit int) []Comment {
	items := getMapSlice(data, "commentlist")
	if items == nil {
		items = getMapSlice(data, "comments")
	}

	var comments []Comment
	for _, item := range items {
		comments = append(comments, Comment{
			ID:         getStringField(item, "commentid", "id"),
			SongID:     songID,
			AuthorName: getStringField(item, "nick", "author_name"),
			Text:       getStringField(item, "rootcomment", "text", "content"),
			LikedCount: getInt(item, "praisenum"),
			CreatedAt:  formatCommentTime(getInt(item, "time")),
		})
		if len(comments) >= limit {
			break
		}
	}
	return comments
}

func formatCommentTime(ts int) string {
	if ts == 0 {
		return ""
	}
	return time.Unix(int64(ts), 0).Format("2006-01-02 15:04:05")
}
