package server

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

type validationResponse struct {
	htmlDataString string
	responseCode   int
	sourceURL      string
	byteRange      string
}

func TestSourceValidationUrl(t *testing.T) {
	var vrList []*validationResponse
	var htmlDataString string
	badSources := []string{"ht/www.example.com", "://www.example.com"}
	responseString := "Bad source string\n"

	serv := NewVimeoService()

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(serv.proxyRequest)

	fmt.Println("====BAD SOURCE URL TEST=====")
	for _, s := range badSources {

		req, err := http.NewRequest("GET", "/", nil)
		if err != nil {
			t.Fatal(err)
		}

		q := req.URL.Query()
		q.Add("s", s)
		req.URL.RawQuery = q.Encode()

		handler.ServeHTTP(rr, req)

		htmlData, err := ioutil.ReadAll(rr.Body)
		if err != nil {
			t.Fatal(err)
		}

		htmlDataString = string(htmlData)

		vr := &validationResponse{
			htmlDataString: htmlDataString,
			responseCode:   rr.Code,
			sourceURL:      s,
			byteRange:      "",
		}

		fmt.Printf("Source URL: %v Response Code: %v Data: %v", vr.sourceURL, vr.responseCode, vr.htmlDataString)

		vrList = append(vrList, vr)

	}

	for _, vr := range vrList {
		if vr.htmlDataString != responseString || vr.responseCode != http.StatusBadRequest {
			t.Errorf("sourceValidation did not catch bad url %v.", vr.sourceURL)
		}
	}
}

func TestSourceValidationByteRange(t *testing.T) {
	var vrList []*validationResponse
	responseString := "Bad byte range\n"
	badRanges := []string{"100-0", "100-", "-100", "-", ""}
	goodSource := "http://storage.googleapis.com/vimeo-test/work-at-vimeo.mp4"

	serv := NewVimeoService()

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(serv.proxyRequest)

	fmt.Println("====BAD BYTE RANGE TEST=====")
	for _, r := range badRanges {

		req, err := http.NewRequest("GET", "/", nil)
		if err != nil {
			t.Fatal(err)
		}

		q := req.URL.Query()
		q.Add("s", goodSource)
		q.Add("range", r)
		req.URL.RawQuery = q.Encode()

		handler.ServeHTTP(rr, req)

		htmlData, err := ioutil.ReadAll(rr.Body)
		if err != nil {
			t.Fatal(err)
		}

		htmlDataString := string(htmlData)

		vr := &validationResponse{
			htmlDataString: htmlDataString,
			responseCode:   rr.Code,
			sourceURL:      goodSource,
			byteRange:      r,
		}

		fmt.Printf("Source URL: %v ByteRange: %v Response Code: %v Data: %v", vr.sourceURL, vr.byteRange, vr.responseCode, vr.htmlDataString)
		vrList = append(vrList, vr)
	}

	for _, vr := range vrList {
		if vr.htmlDataString != responseString || vr.responseCode != http.StatusBadRequest {
			t.Errorf("sourceValidation did not catch bad byte range: %v", vr.byteRange)
		}
	}
}

func TestSourceValidationSourceByteServes(t *testing.T) {
	fmt.Println("====NO ACCEPT-RANGES HEADER TEST=====")
	responseString := "Source does not accept range requests\n"
	goodSource := "http://www.google.com"

	serv := NewVimeoService()

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(serv.proxyRequest)

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	q := req.URL.Query()
	q.Add("s", goodSource)
	req.URL.RawQuery = q.Encode()

	handler.ServeHTTP(rr, req)

	htmlData, err := ioutil.ReadAll(rr.Body)
	if err != nil {
		t.Fatal(err)
	}

	htmlDataString := string(htmlData)

	vr := &validationResponse{
		htmlDataString: htmlDataString,
		responseCode:   rr.Code,
		sourceURL:      goodSource,
	}

	fmt.Printf("Source URL: %v Response Code: %v Data: %v", vr.sourceURL, vr.responseCode, vr.htmlDataString)

	if vr.htmlDataString != responseString || vr.responseCode != http.StatusBadRequest {
		t.Errorf("sourceValidation did not catch byte range header: %v", vr.sourceURL)
	}
}
