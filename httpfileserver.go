package httpfileserver

import (
	"bufio"
	"bytes"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/flate"
	"github.com/klauspost/compress/gzip"
	"github.com/klauspost/compress/zstd"
)

type FileServer struct {
	dir        string
	route      string
	middleware middleware
	cache      sync.Map

	optionDisableCache    bool
	optionMaxBytesPerFile int
}


var mode []string = []string{
	"gzip", "br", "zstd", "deflate",
}

// New returns a new file server that can handle requests for
// files using an in-memory store with gzipping
func New(route, dir string, options ...Option) *FileServer {
	fs := &FileServer{
		dir:                   dir,
		route:                 route,
		optionMaxBytesPerFile: 10000000, // 10 mb
	}
	for _, o := range options {
		o(fs)
	}

	go func() {
		// periodically clean out the sync map of old stuff
		for {
			time.Sleep(1 * time.Minute)
			fs.cache.Range(func(k, v any) bool {
				f, ok := v.(file)
				if !ok {
					return false
				}
				if time.Since(f.date) > 10*time.Minute {
					fs.cache.Delete(k)
				}
				return true
			})
		}
	}()
	return fs
}

// Option is the type all options need to adhere to
type Option func(fs *FileServer)

// OptionNoCache disables the caching
func OptionNoCache(disable bool) Option {
	return func(fs *FileServer) {
		fs.optionDisableCache = disable
	}
}

// OptionMaxBytes sets the maximum number of bytes per file to cache,
// the default is 10 MB
func OptionMaxBytes(optionMaxBytesPerFile int) Option {
	return func(fs *FileServer) {
		fs.optionMaxBytesPerFile = optionMaxBytesPerFile
	}
}

type middleware struct {
	io.Writer
	http.ResponseWriter
	bytesWritten *bytes.Buffer
	numBytes     *int
	overflow     *bool
	maxBytes     int
}

type file struct {
	bytes  []byte
	header http.Header
	date   time.Time
}

type writeCloser struct {
	*bufio.Writer
}

// Close will close the writer
func (wc *writeCloser) Close() error {
	return wc.Flush()
}

// Flush clears all data from cache
func (fs *FileServer) Flush() error {
	fs.cache.Range(func(k, v any) bool {
		_, ok := v.(file)
		if !ok {
			return false
		}
		fs.cache.Delete(k)
		return true
	})
	return nil
}

func (fs *FileServer) Delete(key string) error {
	for _, v := range mode {
		fs.cache.Delete(key + v)
	}
	return nil
}

// Write will have the middleware save the bytes
func (m middleware) Write(b []byte) (int, error) {
	if len(b)+*m.numBytes < m.maxBytes {
		n, _ := m.bytesWritten.Write(b)
		*m.numBytes += n
	} else {
		*m.overflow = true
	}
	return m.Writer.Write(b)
}

// Handle gives a handlerfunc for the file server
func (fs *FileServer) Handle() http.HandlerFunc {
	return fs.ServeHTTP
}

// ServeHTTP is the server of the file server
func (fs *FileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.URL.Path = strings.TrimPrefix(r.URL.Path, fs.route)
	Accept := r.Header.Get("Accept-Encoding")
	// check the sync map using the r.URL.Path and return
	// the gzipped or the standard version
	key := r.URL.Path

	// open from cache if its not disabled
	if !fs.optionDisableCache {
		switch Accept {
		case "gzip", "br", "zstd", "deflate":
			// load the gzipped cache version if available
			fileint, ok := fs.cache.Load(key + Accept)
			if ok {
				file := fileint.(file)
				for k := range file.header {
					for _, v := range file.header[k] {
						w.Header().Set(k, v)
					}
				}
				w.Header().Set("Content-Encoding", Accept)
				w.Write(file.bytes)
				return
			}
		}
		fileint, ok := fs.cache.Load(key)
		// try to load a regular version from the cache
		if ok {
			file := fileint.(file)
			for k := range file.header {
				if k == "Content-Encoding" {
					continue
				}
				for _, v := range file.header[k] {
					if len(v) == 0 {
						continue
					}
					w.Header().Set(k, v)
				}
			}
			switch Accept {
			case "gzip":
				w.Header().Set("Content-Encoding", "gzip")
				var wb bytes.Buffer
				wc := gzip.NewWriter(&wb)
				wc.Write(file.bytes)
				wc.Close()
				w.Write(wb.Bytes())
				file.bytes = wb.Bytes()
				fs.cache.Store(key+"gzip", file)
			case "br":
				w.Header().Set("Content-Encoding", "br")
				var wb bytes.Buffer
				wc := brotli.NewWriter(&wb)
				wc.Write(file.bytes)
				wc.Close()
				w.Write(wb.Bytes())
				file.bytes = wb.Bytes()
				fs.cache.Store(key+"br", file)
			case "zstd":
				w.Header().Set("Content-Encoding", "zstd")
				var wb bytes.Buffer
				enc, _ := zstd.NewWriter(&wb)
				enc.Write(file.bytes)
				enc.Close()
				w.Write(wb.Bytes())
				file.bytes = wb.Bytes()
				fs.cache.Store(key+"br", file)
			case "deflate":
				w.Header().Set("Content-Encoding", "deflate")
				var wb bytes.Buffer
				wc, _ := flate.NewWriter(&wb, -1)
				wc.Write(file.bytes)
				wc.Close()
				w.Write(wb.Bytes())
				file.bytes = wb.Bytes()
				fs.cache.Store(key+"deflate", file)
			default:
				w.Write(file.bytes)
			}
			return
		}
	}

	var wc io.WriteCloser
	switch Accept {
	case "gzip":
		wc = gzip.NewWriter(w)
		w.Header().Set("Content-Encoding", "gzip")
	case "zstd":
		wc, _ = zstd.NewWriter(w)
		w.Header().Set("Content-Encoding", "zstd")
	case "br":
		wc = brotli.NewWriter(w)
		w.Header().Set("Content-Encoding", "br")
	case "deflate":
		wc, _ = flate.NewWriter(w, -1)
		w.Header().Set("Content-Encoding", "deflate")
	default:
		wc = &writeCloser{Writer: bufio.NewWriter(w)}
	}
	defer wc.Close()

	mware := middleware{Writer: wc, ResponseWriter: w, bytesWritten: new(bytes.Buffer), numBytes: new(int), overflow: new(bool), maxBytes: fs.optionMaxBytesPerFile}
	http.FileServer(http.Dir(fs.dir)).ServeHTTP(mware, r)

	// extract bytes written and the header and save it as a file
	// to the sync map using the r.URL.Path
	if !fs.optionDisableCache && !*mware.overflow && !bytes.Equal(mware.bytesWritten.Bytes(), []byte("404 page not found\n")) {
		file := file{
			bytes:  mware.bytesWritten.Bytes(),
			header: w.Header(),
			date:   time.Now(),
		}
		fs.cache.Store(key, file)
	}
}
