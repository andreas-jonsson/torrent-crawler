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
	"crypto/tls"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/jackdanger/collectlinks"
)

const (
	maxDistFromMagnet = 3
	channelSize       = 1024 * 1024
	requestTimeout    = 5
	numThreads        = 16
)

type target struct {
	lnk  *url.URL
	dist int
}

type (
	MagLink struct {
		Title       string
		Lnk, Origin *url.URL
		Ref         int
	}
	MagLinks []MagLink
)

func (a MagLinks) Len() int           { return len(a) }
func (a MagLinks) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a MagLinks) Less(i, j int) bool { return a[i].Ref > a[j].Ref }

var visited struct {
	sync.Mutex
	data map[string]int
}

func Crawl(seeds, domains []string) MagLinks {
	visited.data = make(map[string]int)
	mlinkTable := make(map[string]MagLink)
	inChan := make(chan target, channelSize)
	mlinkChan := make(chan MagLink, numThreads)

	for _, seed := range seeds {
		rawurl, _ := url.Parse(seed)
		inChan <- target{rawurl, 0}
	}

	var outChan chan target
	for i := 0; i < numThreads; i++ {
		outChan = make(chan target, channelSize)
		go crawler(domains, inChan, outChan, mlinkChan)
		inChan = outChan
	}

	go func() {
		for range outChan {
		}
	}()

	interruptChannel := make(chan os.Signal, 1)
	signal.Notify(interruptChannel, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	for {
		select {
		case mlink := <-mlinkChan:
			s := mlink.Lnk.String()
			lnk, ok := mlinkTable[s]

			if ok {
				lnk.Ref++
				mlinkTable[s] = lnk
			} else {
				values := mlink.Lnk.Query()
				title := values.Get("dn")

				mlink.Ref = 1
				mlink.Title = title
				mlinkTable[s] = mlink
			}
		case <-interruptChannel:
			signal.Stop(interruptChannel)
			close(inChan)

			mlinks := make(MagLinks, 0, len(mlinkTable))
			for _, v := range mlinkTable {
				mlinks = append(mlinks, v)
			}
			return mlinks
		}
	}
}

func crawler(domains []string, inTargets <-chan target, outTargets chan<- target, mlinks chan<- MagLink) {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := http.Client{Transport: transport, Timeout: requestTimeout * time.Second}

	for t := range inTargets {
		dist := t.dist + 1
		if dist > maxDistFromMagnet {
			continue
		}

		if len(domains) > 0 {
			found := false
			for _, domain := range domains {
				if strings.Contains(domain, t.lnk.Host) {
					found = true
					break
				}
			}

			if !found {
				continue
			}
		}

		urlStr := t.lnk.String()
		log.Println(urlStr)

		resp, err := client.Get(urlStr)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		links := collectlinks.All(resp.Body)

		visited.Lock()
		for _, link := range links {
			l := fixURL(link, t.lnk)
			if l != nil && l.Scheme == "magnet" {
				mlinks <- MagLink{Lnk: l, Origin: t.lnk}
				dist = 0
			}
		}

		for _, link := range links {
			l := fixURL(link, t.lnk)
			if l != nil {
				s := l.String()
				switch l.Scheme {
				case "http", "https":
					if _, ok := visited.data[s]; !ok {
						select {
						case outTargets <- target{l, dist}:
							visited.data[s] = dist
						default:
							continue
						}
					}
				}
			}
		}
		visited.Unlock()
	}

	close(outTargets)
}

func fixURL(href string, base *url.URL) (ret *url.URL) {
	uri, err := url.Parse(href)
	if err != nil {
		return
	}

	return base.ResolveReference(uri)
}
