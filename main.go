package main

import (
	"encoding/json"
	"io/ioutil"
	"log"

	"github.com/kuronosu/jkanime_scraper/scraper"
)

func main() {
	b, err := json.Marshal(
		scraper.ScrapeAnimes(
			[]string{
				"https://jkanime.net/tensei-shitara-slime-datta-ken/",
				"https://jkanime.net/maoyuu-maou-yuusha/",
				"https://jkanime.net/youjo-senki/",
				"https://jkanime.net/isekai-quartet-2nd-season/",
				"https://jkanime.net/shaman-king-2021/",
				"https://jkanime.net/one-piece/",
			},
		),
	)
	if err != nil {
		panic(err)
	}
	if err = ioutil.WriteFile("animes.json", b, 0644); err != nil {
		log.Println(err)
	}
}
