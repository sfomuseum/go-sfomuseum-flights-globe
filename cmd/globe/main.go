package main

import (
	"context"
	"flag"
	"github.com/mmcloughlin/globe"
	_ "github.com/tidwall/gjson"
	"github.com/whosonfirst/go-whosonfirst-geojson-v2/feature"
	"github.com/whosonfirst/go-whosonfirst-geojson-v2/properties/whosonfirst"
	"github.com/whosonfirst/go-whosonfirst-index"
	"image/color"
	"io"
	"log"
	"sync"
)

func main() {

	size := flag.Int("size", 1024, "...")
	out := flag.String("out", "globe.png", "...")

	lat := flag.Float64("latitude", 37.622096, "...")
	lon := flag.Float64("longitude", -122.384864, "...")

	mode := flag.String("mode", "repo", "...")

	flag.Parse()

	g := globe.New()
	g.DrawGraticule(10.0)
	g.DrawCountryBoundaries()
	// g.DrawLandBoundaries()

	mu := new(sync.RWMutex)

	cb := func(fh io.Reader, ctx context.Context, args ...interface{}) error {

		f, err := feature.LoadGeoJSONFeatureFromReader(fh)

		if err != nil {
			return err
		}

		is_alt := whosonfirst.IsAlt(f)

		if is_alt {
			return nil
		}

		mu.Lock()
		defer mu.Unlock()

		polys, err := f.Polygons()

		if err != nil {
			return err
		}

		for _, poly := range polys {

			ext := poly.ExteriorRing()
			coords := ext.Vertices()

			if len(coords) != 2 {
				continue
			}

			min_x := coords[0].X
			min_y := coords[0].Y
			max_x := coords[1].X
			max_y := coords[1].Y

			g.DrawLine(
				min_y, min_x,
				max_y, max_x,
				globe.Color(color.NRGBA{255, 0, 0, 255}),
			)
		}

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

	g.CenterOn(*lat, *lon)
	g.SavePNG(*out, *size)
}
