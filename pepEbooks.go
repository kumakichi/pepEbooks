package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/sirupsen/logrus"
)

const (
	pageUrl = "http://bp.pep.com.cn/jc/"
)

var (
	download bool
	threads  int
	ch       chan struct{}
	wg       *sync.WaitGroup
)

func init() {
	flag.BoolVar(&download, "d", false, "download files, if not specified, just get links")
	flag.IntVar(&threads, "t", 5, "download threads")
	flag.Parse()

	logrus.SetLevel(logrus.DebugLevel)
}

func main() {
	m, err := genSelectionMap(pageUrl, "li.fl>a")
	if err != nil {
		logrus.Fatal(err)
	}
	ch = make(chan struct{}, threads)
	wg = &sync.WaitGroup{}

	for _, v := range m {
		link, exists := v.Attr("href")
		if !exists {
			continue
		}
		dir := v.Text()
		singlePage(dir, pageUrl+link[2:], wg)
	}
	wg.Wait()
}

func singlePage(dir, link string, wg *sync.WaitGroup) (err error) {
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

		ch <- struct{}{}
		go singleLink(dir, title, pageLink, read, readLink, dl, link+dlLink[2:], ch, wg)
	}
	return
}

func singleLink(dir, title, pageLink, read, readLink, dl, dlLink string, ch chan struct{}, wg *sync.WaitGroup) {
	wg.Add(1)
	defer func() {
		<-ch
		wg.Done()
	}()

	if !download {
		fmt.Println(dir, title, pageLink, read, readLink, dl, dlLink)
		return
	}

	_, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.Mkdir(dir, os.ModePerm)
			if err != nil {
				logrus.Errorf("mkdir %s fail %v", dir, err)
				return
			}
		} else {
			logrus.Errorf("stat %s fail %v", dir, err)
			return
		}
	}

	filePath := dir + "/" + title + ".pdf"
	logrus.Debugf("getting file from %s ...", filePath)
	resp, err := http.Get(dlLink)
	if err != nil {
		logrus.Errorf("get %s fail %v", dlLink, err)
		return
	}
	defer resp.Body.Close()

	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY, os.ModePerm)
	if err != nil {
		logrus.Errorf("open file %s fail %v", filePath, err)
		return
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	if err != nil {
		logrus.Errorf("write file %s fail %v", filePath, err)
	}

	logrus.Debugf("get file %s OK", filePath)
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
