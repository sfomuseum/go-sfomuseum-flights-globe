package main

import (
	"context"
	"flag"
	"github.com/mmcloughlin/globe"
	_ "github.com/tidwall/gjson"
	"image/color"
	"io"
	"log"
	"sync"

	"github.com/whosonfirst/go-whosonfirst-iterate/v2/iterator"
	"github.com/paulmach/orb"	
	"github.com/paulmach/orb/geojson"
)

func main() {

	var iterator_uri string
	
	size := flag.Int("size", 1024, "...")
	out := flag.String("out", "globe.png", "...")

	lat := flag.Float64("latitude", 37.622096, "...")
	lon := flag.Float64("longitude", -122.384864, "...")

	flag.StringVar(&iterator_uri, "iterator-uri", "repo://", "...")

	flag.Parse()

	ctx := context.Background()
	
	g := globe.New()
	g.DrawGraticule(10.0)
	g.DrawCountryBoundaries()
	// g.DrawLandBoundaries()

	mu := new(sync.RWMutex)

	iter_cb := func(ctx context.Context, path string, r io.ReadSeeker, args ...interface{}) error {

		body, err := io.ReadAll(r)

		if err != nil {
			return err
		}
		
		f, err := geojson.UnmarshalFeature(body)

		if err != nil {
			return err
		}

		geom := f.Geometry
		
		mu.Lock()
		defer mu.Unlock()

		switch geom.GeoJSONType() {
		case "MultiPoint":

			/*
			mp := geom.(orb.MultiPoint)

			g.DrawLine(
				mp[0][1], mp[0][0],
				mp[1][1], mp[1][0],				
				globe.Color(color.NRGBA{255, 0, 0, 255}),
			)
			*/
			
		case "LineString":

			ls := geom.(orb.LineString)
			count := len(ls)
			
			for i := 1; i < count; i++ {
				
				g.DrawLine(
					ls[i-1][1], ls[i-1][0],
					ls[i][1], ls[i][0],				
					globe.Color(color.NRGBA{255, 0, 0, 255}),
				)
				
			}
			
		case "MultiLineString":

			mls := geom.(orb.MultiLineString)

			for _, ls := range mls {
				
				count := len(ls)
				
				for i := 1; i < count; i++ {
					
					g.DrawLine(
						ls[i-1][1], ls[i-1][0],
						ls[i][1], ls[i][0],				
						globe.Color(color.NRGBA{255, 0, 255, 0}),
					)
					
				}
			}
			
		default:
			log.Println("UNSUPPORTED", geom.GeoJSONType())
		}
		
		return nil
	}

	iter, err := iterator.NewIterator(ctx, iterator_uri, iter_cb)

	if err != nil {
		log.Fatal(err)
	}

	uris := flag.Args()
	err = iter.IterateURIs(ctx, uris...)

	if err != nil {
		log.Fatal(err)
	}

	g.CenterOn(*lat, *lon)
	g.SavePNG(*out, *size)
}
