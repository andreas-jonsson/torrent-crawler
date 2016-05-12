/*
Copyright (C) 2016 Andreas T Jonsson

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"sort"
)

// https://torrentfreak.com/top-10-most-popular-torrent-sites-of-2016-160102/
var torrentSites = []string{
	"https://kat.cr",
	"https://thepiratebay.se",
	"http://extratorrent.cc",
	"http://www.torrentz.eu",
	"http://rarbg.to",
	"http://1337x.to",
	"http://www.torrenthound.com",
	"http://yts.ag",
	"http://torrentdownloads.me",
}

func main() {
	links := Crawl(torrentSites, torrentSites)
	sort.Sort(links)

	var buf bytes.Buffer
	fmt.Fprintln(&buf, "<!DOCTYPE html><html><body>")
	for _, mag := range links {
		fmt.Fprintf(&buf, `(%d) %s <a href="%s">%s</a><br>`, mag.Ref, mag.Origin.Host, mag.Lnk.String(), mag.Title)
		fmt.Fprintln(&buf)
	}
	fmt.Fprintln(&buf, "</body></html>")

	ioutil.WriteFile("torrents.html", buf.Bytes(), 0644)
	if data, err := json.MarshalIndent(links, "", "\t"); err == nil {
		ioutil.WriteFile("torrents.json", data, 0644)
	}

	fmt.Println("Result is saved!")
}
