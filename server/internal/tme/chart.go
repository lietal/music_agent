package tme

import (
	"context"
	"fmt"
)

func (c *Client) GetChartCategories(ctx context.Context) ([]Chart, error) {
	resp, err := c.Call(ctx, map[string]MusicuSubRequest{
		"req_0": {
			Module: "music.toplist.TopList",
			Method: "GetAll",
			Param:  map[string]any{},
		},
	})
	if err != nil {
		return nil, err
	}

	sub, ok := resp.Req["req_0"]
	if !ok || sub.Code != 0 {
		return nil, fmt.Errorf("chart categories failed: code=%d", sub.Code)
	}

	var charts []Chart
	for _, group := range getMapSlice(sub.Data, "group") {
		for _, item := range getMapSlice(group, "toplist") {
			charts = append(charts, Chart{
				ID:              "qqmusic:chart:" + getString(item, "topId"),
				Name:            getString(item, "title"),
				Description:     getString(item, "intro"),
				UpdateFrequency: getString(item, "updateFrequency"),
				ArtworkURL:      buildArtworkURL(getString(item, "pic")),
			})
		}
	}
	return charts, nil
}

func (c *Client) GetChartDetail(ctx context.Context, chartID string, limit, page int) ([]Song, error) {
	resp, err := c.Call(ctx, map[string]MusicuSubRequest{
		"req_0": {
			Module: "music.toplist.TopList",
			Method: "GetDetail",
			Param: map[string]any{
				"topId":       extractMID(chartID),
				"num":         limit,
				"page":        page,
				"platform":    "android",
			},
		},
	})
	if err != nil {
		return nil, err
	}

	sub, ok := resp.Req["req_0"]
	if !ok || sub.Code != 0 {
		return nil, fmt.Errorf("chart detail failed: code=%d", sub.Code)
	}

	return c.parseSongs(sub.Data), nil
}

func (c *Client) GetRecommendTracks(ctx context.Context, recType string, limit, page int) ([]Song, error) {
	switch recType {
	case "radar":
		return c.getRadarTracks(ctx, limit)
	default:
		return c.getGuessTracks(ctx, limit)
	}
}

func (c *Client) getGuessTracks(ctx context.Context, limit int) ([]Song, error) {
	resp, err := c.Call(ctx, map[string]MusicuSubRequest{
		"req_0": {
			Module: "music.radioProxy.MbTrackRadioSvr",
			Method: "get_radio_track",
			Param: map[string]any{
				"platform": "android",
				"version":  14090008,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	sub, ok := resp.Req["req_0"]
	if !ok || sub.Code != 0 {
		return nil, fmt.Errorf("recommend tracks failed: code=%d", sub.Code)
	}

	return c.parseSongs(sub.Data), nil
}

func (c *Client) getRadarTracks(ctx context.Context, limit int) ([]Song, error) {
	resp, err := c.Call(ctx, map[string]MusicuSubRequest{
		"req_0": {
			Module: "music.recommend.TrackRelationServer",
			Method: "GetRadarSong",
			Param: map[string]any{
				"num": limit,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	sub, ok := resp.Req["req_0"]
	if !ok || sub.Code != 0 {
		return nil, fmt.Errorf("radar tracks failed: code=%d", sub.Code)
	}

	return c.parseSongs(sub.Data), nil
}
