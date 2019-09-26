package main

// this is work in progress - the point here is not so much to fetch data with
// go-whosonfirst-index but to think about how/what data we need to get in order
// to make a pretty map and how, for example the globe may not actually be the
// best interface... (20190925/thisisaaronland)

import (
	"context"
	"flag"
	"github.com/mmcloughlin/globe"
	"github.com/skelterjohn/geom"
	"github.com/tidwall/gjson"
	"github.com/whosonfirst/go-whosonfirst-geojson-v2"
	"github.com/whosonfirst/go-whosonfirst-geojson-v2/feature"
	"github.com/whosonfirst/go-whosonfirst-geojson-v2/properties/whosonfirst"
	"github.com/whosonfirst/go-whosonfirst-index"
	"github.com/whosonfirst/go-whosonfirst-readwrite/reader"
	"github.com/whosonfirst/go-whosonfirst-uri"
	"image/color"
	"io"
	"log"
	"math"
	"sync"
)

func LoadFeatureFromReader(rd reader.Reader, id int64) (geojson.Feature, error) {

	rel_path, err := uri.Id2RelPath(id)

	if err != nil {
		return nil, err
	}

	fh, err := rd.Read(rel_path)

	if err != nil {
		return nil, err
	}

	feature, err := feature.LoadWOFFeatureFromReader(fh)

	if err != nil {
		return nil, err
	}

	return feature, nil
}

func main() {

	mode := flag.String("mode", "repo", "")
	key := flag.String("key", "", "")

	// make this a multi-value flag

	value := flag.String("value", "", "")

	flag.Parse()

	switch *key {
	case "tailnumber", "aircraft":
		// pass
	default:
		log.Fatal("Invalid key")
	}

	fs_root := "/usr/local/data/sfomuseum-data-whosonfirst/data"
	rd, err := reader.NewFSReader(fs_root)

	if err != nil {
		log.Fatal(err)
	}

	sfo := int64(102527513)

	others := new(sync.Map)

	cb := func(fh io.Reader, ctx context.Context, args ...interface{}) error {

		f, err := feature.LoadGeoJSONFeatureFromReader(fh)

		if err != nil {
			return err
		}

		is_alt := whosonfirst.IsAlt(f)

		if is_alt {
			return nil
		}

		// this logic will need to be updated once -value is a multi-value flag

		match := false

		switch *key {
		case "aircraft":

			rsp := gjson.GetBytes(f.Bytes(), "properties.icao:aircraft")

			if rsp.Exists() && rsp.String() == *value {
				match = true
			}

		case "tailnumber":

			tail_rsp := gjson.GetBytes(f.Bytes(), "properties.swim:tail_number")

			if tail_rsp.Exists() && tail_rsp.String() == *value {
				match = true
			}

		default:
			// this should never happen
		}

		if !match {
			return nil
		}

		arrival_rsp := gjson.GetBytes(f.Bytes(), "properties.sfomuseum:arrival_id")

		if !arrival_rsp.Exists() {
			return nil
		}

		departure_rsp := gjson.GetBytes(f.Bytes(), "properties.sfomuseum:departure_id")

		if !departure_rsp.Exists() {
			return nil
		}

		arrival_id := arrival_rsp.Int()
		departure_id := departure_rsp.Int()

		var other int64

		if arrival_id == sfo {
			other = departure_id
		} else {
			other = arrival_id
		}

		others.Store(other, true)
		return nil
	}

	idx, err := index.NewIndexer(*mode, cb)

	if err != nil {
		log.Fatal(err)
	}

	paths := flag.Args()
	err = idx.IndexPaths(paths)

	if err != nil {
		log.Fatal(err)
	}

	sfo_feature, err := LoadFeatureFromReader(rd, sfo)

	if err != nil {
		log.Fatal(err)
	}

	sfo_centroid, err := whosonfirst.Centroid(sfo_feature)

	if err != nil {
		log.Fatal(err)
	}

	sfo_coords := sfo_centroid.Coord()

	g := globe.New()
	g.DrawGraticule(10.0)
	g.DrawCountryBoundaries()

	coords := make([]geom.Coord, 0)

	others.Range(func(key interface{}, value interface{}) bool {

		other_id := key.(int64)

		feature, err := LoadFeatureFromReader(rd, other_id)

		if err != nil {
			log.Println(err)
			return false
		}

		// TO DO: flag airports that are "visiting" null islan

		centroid, err := whosonfirst.Centroid(feature)

		if err != nil {
			log.Println(err)
			return false
		}

		coords = append(coords, centroid.Coord())
		return true
	})

	min_y := sfo_coords.Y
	min_x := sfo_coords.X
	max_y := sfo_coords.Y
	max_x := sfo_coords.X

	for _, c := range coords {
		min_y = math.Min(min_y, c.Y)
		min_x = math.Min(min_x, c.X)
		max_y = math.Max(max_y, c.Y)
		max_x = math.Max(max_x, c.X)
	}

	/*
		g.DrawRect(
			min_y, min_x,
			max_y, max_x,
			globe.Color(color.NRGBA{255, 0, 0, 255}),
		)
	*/

	for _, c := range coords {
		g.DrawLine(
			sfo_coords.Y, sfo_coords.X,
			c.Y, c.X,
			globe.Color(color.NRGBA{255, 0, 0, 255}),
		)
	}

	green := color.NRGBA{0x00, 0x64, 0x3c, 192}

	g.DrawDot(sfo_coords.Y, sfo_coords.X, 0.05, globe.Color(green))

	for _, c := range coords {
		g.DrawDot(c.Y, c.X, 0.05, globe.Color(green))
	}

	bounds := geom.NilRect()
	bounds.Min.X = min_x
	bounds.Min.Y = min_y
	bounds.Max.X = max_x
	bounds.Max.Y = max_y

	center := bounds.Center()
	g.CenterOn(center.Y, center.X)

	g.SavePNG("test.png", 2048)

}
