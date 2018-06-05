package main

import (
	"fmt"
	"log"

	"github.com/sniperkit/go-tablib"
	//"github.com/sniperkit/xutil/plugin/debug/pp"
)

var entry struct {
	URL string `name:"url"`
	// Valid bool   `name:"-"`
	// Done  bool   `name:"-"`
}

func urlLen(row []interface{}) interface{} {
	return len(row[0].(string))
}

func priority(row []interface{}) interface{} {
	return 0.8
}

func loc(row []interface{}) interface{} {
	return row[0].(string)
}

func lastmod(row []interface{}) interface{} {
	return "2005-01-01"
}

func changefreq(row []interface{}) interface{} {
	return "monthly"
}

func main() {

	f := "./shared/data/sitemap/sitemap.txt"
	ds, err := tablib.LoadFileCSV(f)
	if err != nil {
		log.Fatal(err)
	}

	ds.AppendDynamicColumn("priority", priority)
	// ds.AppendDynamicColumn("loc", loc)
	ds.AppendDynamicColumn("lastmod", lastmod)
	ds.AppendDynamicColumn("changefreq", changefreq)

	xml, err := ds.XML()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(xml)

}
