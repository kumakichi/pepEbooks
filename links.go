package main

import (
	"fmt"
	"net/http"

	"github.com/PuerkitoBio/goquery"
	"github.com/sirupsen/logrus"
)

const (
	pageUrl = "http://bp.pep.com.cn/jc/"
)

func main() {
	m, err := genSelectionMap(pageUrl, "li.fl>a")
	if err != nil {
		logrus.Fatal(err)
	}

	for _, v := range m {
		link, exists := v.Attr("href")
		if !exists {
			continue
		}
		dir := v.Text()
		singlePage(dir, pageUrl+link[2:])
	}
}

func singlePage(dir, link string) (err error) {
	m, err := genSelectionMap(link, "li.fl")
	if err != nil {
		return
	}

	for _, v := range m {
		pageLink, title, err := nodeAttrAndText(v, "h6>a", "href")
		if err != nil {
			logrus.Errorf("find title fail:%v", err)
			continue
		}
		readLink, read, err := nodeAttrAndText(v, ".btn_type_dy", "href")
		if err != nil {
			logrus.Errorf("find read fail:%v", err)
			continue
		}
		dlLink, dl, err := nodeAttrAndText(v, ".btn_type_dl", "href")
		if err != nil {
			logrus.Errorf("find dl fail:%v", err)
			continue
		}

		fmt.Println(dir, title, pageLink, read, readLink, dl, link+dlLink[2:])
	}
	return
}

func nodeAttrAndText(s *goquery.Selection, selector, attrKey string) (attr, text string, err error) {
	node := s.Find(selector)
	if node == nil {
		err = fmt.Errorf("node not found")
		return
	}

	attr, exists := node.Attr(attrKey)
	if !exists {
		err = fmt.Errorf("attr %s not found", attrKey)
		return
	}

	text = node.Text()
	return
}

func genSelectionMap(link, selector string) (m map[int]*goquery.Selection, err error) {
	resp, err := http.Get(link)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return
	}

	m = make(map[int]*goquery.Selection)
	doc.Find(selector).Each(func(i int, selection *goquery.Selection) {
		m[i] = selection
	})
	return
}
