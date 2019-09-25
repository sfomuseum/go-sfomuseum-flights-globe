package main

import (
	"context"
	"flag"
	"github.com/mmcloughlin/globe"	
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
	tailnumber := flag.String("tailnumber", "", "")

	flag.Parse()

	if *tailnumber == "" {
		log.Fatal("Missing tail number")
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

		tail_rsp := gjson.GetBytes(f.Bytes(), "properties.swim:tail_number")

		if !tail_rsp.Exists(){
			return nil
		}

		if tail_rsp.String() != *tailnumber {
			return nil
		}

		arrival_rsp := gjson.GetBytes(f.Bytes(), "properties.sfomuseum:arrival_id")

		if !arrival_rsp.Exists(){
			return nil
		}
		
		departure_rsp := gjson.GetBytes(f.Bytes(), "properties.sfomuseum:departure_id")

		if !departure_rsp.Exists(){
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

	log.Println(sfo, sfo_coords)
	
	others.Range(func(key interface{}, value interface{}) bool {

		other_id := key.(int64)

		feature, err := LoadFeatureFromReader(rd, other_id)

		if err != nil {
			log.Println(err)
			return false
		}

		centroid, err := whosonfirst.Centroid(feature)

		if err != nil {
			log.Println(err)
			return false
		}

		coords := centroid.Coord()

			g.DrawLine(
				sfo_coords.Y, sfo_coords.X,
				coords.Y, coords.X,
				globe.Color(color.NRGBA{255, 0, 0, 255}),
			)
		
		return true
	})

	g.CenterOn(sfo_coords.Y, sfo_coords.X)
	g.SavePNG("test.png", 2048)
	
}
