/*
 * jQuery File Upload Plugin GAE Go Example
 * https://github.com/blueimp/jQuery-File-Upload
 *
 * Copyright 2011, Sebastian Tschan
 * https://blueimp.net
 *
 * Licensed under the MIT license:
 * http://www.opensource.org/licenses/MIT
 */

package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/disintegration/gift"
	/*
		"golang.org/x/net/context"
		"google.golang.org/appengine"
		"google.golang.org/appengine/memcache"
	*/
	"hash/crc32"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	WEBSITE       = "http://192.168.99.100:8000/"
	HOST          = "192.168.99.100:8080"
	MEMCACHE      = "http://172.17.0.4:1978/"
	MIN_FILE_SIZE = 1 // bytes
	// Max file size is memcache limit (1MB) minus key size minus overhead:
	MAX_FILE_SIZE     = 999000 // bytes
	IMAGE_TYPES       = "image/(gif|p?jpeg|(x-)?png)"
	ACCEPT_FILE_TYPES = IMAGE_TYPES
	THUMB_MAX_WIDTH   = 80
	THUMB_MAX_HEIGHT  = 80
	EXPIRATION_TIME   = 300 // seconds
	// If empty, only allow redirects to the referer protocol+host.
	// Set to a regexp string for custom pattern matching:
	REDIRECT_ALLOW_TARGET = ""
)

var (
	imageTypes      = regexp.MustCompile(IMAGE_TYPES)
	acceptFileTypes = regexp.MustCompile(ACCEPT_FILE_TYPES)
	thumbSuffix     = "." + fmt.Sprint(THUMB_MAX_WIDTH) + "x" +
		fmt.Sprint(THUMB_MAX_HEIGHT)
)

func escape(s string) string {
	return strings.Replace(url.QueryEscape(s), "+", "%20", -1)
}

func extractKey(r *http.Request) string {
	// Use RequestURI instead of r.URL.Path, as we need the encoded form:
	path := strings.Split(r.RequestURI, "?")[0]
	// Also adjust double encoded slashes:
	return strings.Replace(path[1:], "%252F", "%2F", -1)
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

type FileInfo struct {
	Key          string `json:"-"`
	ThumbnailKey string `json:"-"`
	Url          string `json:"url,omitempty"`
	ThumbnailUrl string `json:"thumbnailUrl,omitempty"`
	Name         string `json:"name"`
	Type         string `json:"type"`
	Size         int64  `json:"size"`
	Error        string `json:"error,omitempty"`
	DeleteUrl    string `json:"deleteUrl,omitempty"`
	DeleteType   string `json:"deleteType,omitempty"`
}

func (fi *FileInfo) ValidateType() (valid bool) {
	if acceptFileTypes.MatchString(fi.Type) {
		return true
	}
	fi.Error = "Filetype not allowed"
	return false
}

func (fi *FileInfo) ValidateSize() (valid bool) {
	if fi.Size < MIN_FILE_SIZE {
		fi.Error = "File is too small"
	} else if fi.Size > MAX_FILE_SIZE {
		fi.Error = "File is too big"
	} else {
		return true
	}
	return false
}

func (fi *FileInfo) CreateUrls(r *http.Request) {
	u := &url.URL{
		Scheme: r.URL.Scheme,
		//Host:   appengine.DefaultVersionHostname(c),
		Host: HOST,
		Path: "/",
	}
	uString := u.String()
	fi.Url = uString + fi.Key
	fi.DeleteUrl = fi.Url
	fi.DeleteType = "DELETE"
	if fi.ThumbnailKey != "" {
		fi.ThumbnailUrl = uString + fi.ThumbnailKey
	}
}

func (fi *FileInfo) SetKey(checksum uint32) {
	fi.Key = escape(string(fi.Type)) + "-" +
		escape(fmt.Sprint(checksum)) + "-" +
		escape(string(fi.Name))
}

//func (fi *FileInfo) createThumb(buffer *bytes.Buffer, c context.Context) {
func (fi *FileInfo) createThumb(buffer *bytes.Buffer) {
	if imageTypes.MatchString(fi.Type) {
		src, _, err := image.Decode(bytes.NewReader(buffer.Bytes()))
		check(err)
		filter := gift.New(gift.ResizeToFit(
			THUMB_MAX_WIDTH,
			THUMB_MAX_HEIGHT,
			gift.LanczosResampling,
		))
		dst := image.NewNRGBA(filter.Bounds(src.Bounds()))
		filter.Draw(dst, src)
		buffer.Reset()
		bWriter := bufio.NewWriter(buffer)
		switch fi.Type {
		case "image/jpeg", "image/pjpeg":
			err = jpeg.Encode(bWriter, dst, nil)
		case "image/gif":
			err = gif.Encode(bWriter, dst, nil)
		default:
			err = png.Encode(bWriter, dst)
		}
		check(err)
		bWriter.Flush()
		thumbnailKey := fi.Key + thumbSuffix + filepath.Ext(fi.Name)
		/*
			item := &memcache.Item{
				Key:   thumbnailKey,
				Value: buffer.Bytes(),
			}
			err = memcache.Set(c, item)
		*/
		err = memcacheSet(thumbnailKey, buffer.Bytes())
		fmt.Println("Key=",thumbnailKey)
		check(err)
		fi.ThumbnailKey = thumbnailKey
	}
}

func handleUpload(r *http.Request, p *multipart.Part) (fi *FileInfo) {
	fi = &FileInfo{
		Name: p.FileName(),
		Type: p.Header.Get("Content-Type"),
	}
	if !fi.ValidateType() {
		return
	}
	defer func() {
		if rec := recover(); rec != nil {
			log.Println(rec)
			fi.Error = rec.(error).Error()
		}
	}()
	var buffer bytes.Buffer
	hash := crc32.NewIEEE()
	mw := io.MultiWriter(&buffer, hash)
	lr := &io.LimitedReader{R: p, N: MAX_FILE_SIZE + 1}
	_, err := io.Copy(mw, lr)
	check(err)
	fi.Size = MAX_FILE_SIZE + 1 - lr.N
	if !fi.ValidateSize() {
		return
	}
	fi.SetKey(hash.Sum32())
	/*
		item := &memcache.Item{
			Key:   fi.Key,
			Value: buffer.Bytes(),
		}
		context := appengine.NewContext(r)
		err = memcache.Set(context, item)
	*/
	err = memcacheSet(fi.Key, buffer.Bytes())
	check(err)
	//fi.createThumb(&buffer, context)
	fi.createThumb(&buffer)
	fi.CreateUrls(r)
	return
}

func getFormValue(p *multipart.Part) string {
	var b bytes.Buffer
	io.CopyN(&b, p, int64(1<<20)) // Copy max: 1 MiB
	return b.String()
}

func handleUploads(r *http.Request) (fileInfos []*FileInfo) {
	fileInfos = make([]*FileInfo, 0)
	mr, err := r.MultipartReader()
	check(err)
	r.Form, err = url.ParseQuery(r.URL.RawQuery)
	check(err)
	part, err := mr.NextPart()
	for err == nil {
		if name := part.FormName(); name != "" {
			if part.FileName() != "" {
				fileInfos = append(fileInfos, handleUpload(r, part))
			} else {
				r.Form[name] = append(r.Form[name], getFormValue(part))
			}
		}
		part, err = mr.NextPart()
	}
	return
}

func validateRedirect(r *http.Request, redirect string) bool {
	if redirect != "" {
		var redirectAllowTarget *regexp.Regexp
		if REDIRECT_ALLOW_TARGET != "" {
			redirectAllowTarget = regexp.MustCompile(REDIRECT_ALLOW_TARGET)
		} else {
			referer := r.Referer()
			if referer == "" {
				return false
			}
			refererUrl, err := url.Parse(referer)
			if err != nil {
				return false
			}
			redirectAllowTarget = regexp.MustCompile("^" + regexp.QuoteMeta(
				refererUrl.Scheme+"://"+refererUrl.Host+"/",
			))
		}
		return redirectAllowTarget.MatchString(redirect)
	}
	return false
}

func get(w http.ResponseWriter, r *http.Request) {
	fmt.Println("In get req")
	if r.URL.Path == "/" {
		http.Redirect(w, r, WEBSITE, http.StatusFound)
		return
	}
	// Use RequestURI instead of r.URL.Path, as we need the encoded form:
	key := extractKey(r)
	fmt.Println("Key get=",key)
	parts := strings.Split(key, "/")
	fmt.Println("Parts get=",parts)
	if len(parts) == 1 {
		/*
			context := appengine.NewContext(r)
			item, err := memcache.Get(context, key)
		*/
		fmt.Println("Get key=",key)
		item, err := memcacheGet(key)
		if err == nil {
			w.Header().Add("X-Content-Type-Options", "nosniff")
			contentType, _ := url.QueryUnescape(parts[0])
			if !imageTypes.MatchString(contentType) {
				contentType = "application/octet-stream"
			}
			w.Header().Add("Content-Type", contentType)
			w.Header().Add(
				"Cache-Control",
				fmt.Sprintf("public,max-age=%d", EXPIRATION_TIME),
			)
			w.Write([]byte(item))
			return
		}
	}
	http.Error(w, "404 Not Found", http.StatusNotFound)
}

func post(w http.ResponseWriter, r *http.Request) {
	result := make(map[string][]*FileInfo, 1)
	result["files"] = handleUploads(r)
	b, err := json.Marshal(result)
	check(err)
	if redirect := r.FormValue("redirect"); validateRedirect(r, redirect) {
		if strings.Contains(redirect, "%s") {
			redirect = fmt.Sprintf(
				redirect,
				escape(string(b)),
			)
		}
		http.Redirect(w, r, redirect, http.StatusFound)
		return
	}
	w.Header().Set("Cache-Control", "no-cache")
	jsonType := "application/json"
	if strings.Index(r.Header.Get("Accept"), jsonType) != -1 {
		w.Header().Set("Content-Type", jsonType)
	}
	fmt.Fprintln(w, string(b))
}

func delete(w http.ResponseWriter, r *http.Request) {
	key := extractKey(r)
	parts := strings.Split(key, "/")
	if len(parts) == 1 {
		result := make(map[string]bool, 1)
		/*
			context := appengine.NewContext(r)
			err := memcache.Delete(context, key)
		*/
		err := memcacheDelete(key)
		if err == nil {
			result[key] = true
			contentType, _ := url.QueryUnescape(parts[0])
			if imageTypes.MatchString(contentType) {
				//thumbnailKey := key + thumbSuffix + filepath.Ext(parts[2])
				thumbnailKey := key + thumbSuffix + filepath.Ext(parts[0])
				//err := memcache.Delete(context, thumbnailKey)
				err := memcacheDelete(thumbnailKey)
				if err == nil {
					result[thumbnailKey] = true
				}
			}
		}
		w.Header().Set("Content-Type", "application/json")
		b, err := json.Marshal(result)
		check(err)
		fmt.Fprintln(w, string(b))
	} else {
		http.Error(w, "405 Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handle(w http.ResponseWriter, r *http.Request) {
	params, err := url.ParseQuery(r.URL.RawQuery)
	check(err)
	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.Header().Add(
		"Access-Control-Allow-Methods",
		"OPTIONS, HEAD, GET, POST, DELETE",
	)
	w.Header().Add(
		"Access-Control-Allow-Headers",
		"Content-Type, Content-Range, Content-Disposition",
	)
	switch r.Method {
	case "OPTIONS", "HEAD":
		return
	case "GET":
		get(w, r)
	case "POST":
		if len(params["_method"]) > 0 && params["_method"][0] == "DELETE" {
			delete(w, r)
		} else {
			post(w, r)
		}
	case "DELETE":
		delete(w, r)
	default:
		http.Error(w, "501 Not Implemented", http.StatusNotImplemented)
	}
}

func memcacheGet(key string) (string, error) {
	URL := fmt.Sprintf("%s%s", MEMCACHE, key)
	response, err := http.Get(URL)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	contents, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", err
	}
	return string(contents), nil
}
func memcacheSet(key string, value []byte) error {
	URL := fmt.Sprintf("%s%s", MEMCACHE, key)
	client := &http.Client{}
	req, err := http.NewRequest("PUT", URL, bytes.NewBuffer(value))
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	fmt.Println(resp.Status)
	return nil
}

func memcacheDelete(key string) error {
	URL := fmt.Sprintf("%s%s", MEMCACHE, key)
	client := &http.Client{}
	req, err := http.NewRequest("DELETE", URL, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	//fmt.Println(resp.Status)
	return nil
}

func main() {
	http.HandleFunc("/", handle)
	http.ListenAndServe(":8080", nil)
}
