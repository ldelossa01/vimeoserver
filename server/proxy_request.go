package server

import (
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/ldelossa/vimeoserver/cache"
)

// Constant declaration
var (
	ErrInvalidRange  = errors.New("Invalid range")
	ErrInvalidSource = errors.New("Invalid source")
)

func (s *VimeoService) proxyRequest(w http.ResponseWriter, r *http.Request) {
	var ranges []int // Slice holding provided ranges if provided
	var err error
	var respBytes []byte       // Byte array holding response from origin
	var rangeProvided bool     // Is range provided?
	var byteRangeString string // String containing passed byte range URL param

	// parse params out of url
	params := r.URL.Query()

	// validate range header if present, set appropriate variables
	if s, ok := params["range"]; ok {
		if ranges, err = rangeValidation(s[0], w); err != nil {
			rangeProvided = false
			return
		}
		rangeProvided = true
		byteRangeString = s[0]
	}

	// we need a source address in our request parameters
	if _, ok := params["s"]; !ok {
		http.Error(w, "Source string not provided", http.StatusBadRequest)
		return
	}

	// Test url is valid
	sourceURL := strings.Trim(params["s"][0], "\"")
	if err = s.sourceValidation(sourceURL, w); err != nil {
		return
	}

	// If range provided, attempt cache serve, store array response in respBytes
	if rangeProvided {

		// Attempt cache lookup
		respBytes, err = s.cache.Get(ranges[0], ranges[1], sourceURL)

		// If found in cache, return respBytes to ResponseWriter and return
		if err != cache.ErrCacheMiss {
			w.Write(respBytes)
			return
		}

		// If cache miss: create new request, add appropraite header, place into cache, and return respBytes
		req, err := http.NewRequest("GET", sourceURL, nil)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		// Add Range header to new request
		req.Header.Add("Range", "bytes="+strings.Trim(byteRangeString, "\""))

		// Perform request, close body on function close, handle errors
		resp, err := s.httpClient.Do(req)
		defer resp.Body.Close()
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}

		// do not cache non 206 codes
		if resp.StatusCode == 206 {
			// Copy bytes from resp.Body to respBytes buffer to place in cache
			respBytes, err = ioutil.ReadAll(resp.Body)
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}

			// Asyc place bytes into cache
			go s.cache.Put(ranges[0], ranges[1], respBytes, sourceURL)

			// Write respBytes to ResponseWriter and return
			w.Write(respBytes)
			return
		}
		// If no range provided, simpley proxy the response
	} else {

		req, err := http.NewRequest("GET", sourceURL, nil)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Perform request, close body on function close, handle errors
		resp, err := s.httpClient.Do(req)
		defer resp.Body.Close()
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		respBytes, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		w.Write(respBytes)
	}
}

// Confirm that the source is valid
func (s *VimeoService) sourceValidation(sourceURL string, w http.ResponseWriter) error {
	if _, err := url.ParseRequestURI(sourceURL); err != nil {
		http.Error(w, "Bad source string", http.StatusBadRequest)
		return ErrInvalidSource
	}

	// Determine if source address supports range requests
	resp, err := s.httpClient.Head(sourceURL)
	if err != nil {
		http.Error(w, "Bad source string, does not support range requests", http.StatusBadRequest)
		return ErrInvalidSource
	}

	if _, ok := resp.Header["Accept-Ranges"]; !ok {
		http.Error(w, "Source does not accept range requests", http.StatusBadRequest)
		return ErrInvalidSource
	}

	for _, b := range resp.Header["Accept-Ranges"] {
		if strings.ToLower(b) == "bytes" {
			break
		} else {
			http.Error(w, "Source does not accept range requests", http.StatusBadRequest)
			return ErrInvalidSource
		}
	}

	// Proxy source content type to caller
	w.Header().Set("Content-Type", resp.Header["Content-Type"][0])
	return nil
}

// confirm that range value is valid
func rangeValidation(brange string, w http.ResponseWriter) ([]int, error) {
	tokens := strings.Split(brange, "-")

	// Range should always have two values
	if len(tokens) != 2 {
		http.Error(w, "Bad byte range", http.StatusBadRequest)
		return nil, ErrInvalidRange
	}

	// Convert strings to ints
	r1, err := strconv.Atoi(strings.Trim(tokens[0], "\""))
	if err != nil {
		http.Error(w, "Bad byte range", http.StatusBadRequest)
		return nil, ErrInvalidRange
	}
	r2, err := strconv.Atoi(strings.Trim(tokens[1], "\""))
	if err != nil {
		http.Error(w, "Bad byte range", http.StatusBadRequest)
		return nil, ErrInvalidRange
	}

	// Start range should always be less then end
	if r1 > r2 {
		http.Error(w, "Bad byte range", http.StatusBadRequest)
		return nil, ErrInvalidRange
	}

	return []int{r1, r2}, nil
}
